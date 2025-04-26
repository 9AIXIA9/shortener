package cachex

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"shortener/pkg/errorx"
	"strconv"
	"time"
)

func NewRedisSequenceCache(rdb *redis.Redis, keySequenceID string, keySequenceState string) SequenceCache {
	return &redisSequenceCache{
		rdb:              rdb,
		keySequenceID:    keySequenceID,
		keySequenceState: keySequenceState,
	}
}

type redisSequenceCache struct {
	rdb              *redis.Redis
	keySequenceID    string
	keySequenceState string
}

func (c *redisSequenceCache) GetSingleID(ctx context.Context) (uint64, error) {
	val, err := c.rdb.LpopCtx(ctx, c.keySequenceID)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, errorx.New(errorx.CodeNotFound, "no sequence found in redis")
		}
		return 0, errorx.NewWithCause(errorx.CodeCacheError, "failed to get sequence from redis", err)
	}

	id, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, errorx.NewWithCause(errorx.CodeSystemError, "failed to parse the sequence id", err)
	}

	return id, nil
}

func (c *redisSequenceCache) FillIDs(ctx context.Context, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}

	// Use pipeline for batch writing
	err := c.rdb.PipelinedCtx(ctx, func(pipe redis.Pipeliner) error {
		for _, id := range ids {
			if err := pipe.RPush(ctx, c.keySequenceID, strconv.FormatUint(id, 10)).Err(); err != nil {
				return errorx.NewWithCause(errorx.CodeCacheError, "failed to push sequence id", err)
			}
		}
		// Update state to indicate last update time
		if err := pipe.Set(ctx, c.keySequenceState, time.Now().Unix(), 0).Err(); err != nil {
			return errorx.NewWithCause(errorx.CodeCacheError, "failed to set sequence state", err)
		}

		return nil
	})

	if err != nil {
		return errorx.NewWithCause(errorx.CodeCacheError, "failed to fill ids to redis", err)
	}

	return nil
}

func (c *redisSequenceCache) IsOK(ctx context.Context) bool {
	return c.rdb.PingCtx(ctx)
}

func (c *redisSequenceCache) IsLessThanThreshold(ctx context.Context, threshold int) (bool, error) {
	// Get the length of the list
	length, err := c.rdb.LlenCtx(ctx, c.keySequenceID)
	if err != nil {
		return false, errorx.NewWithCause(errorx.CodeCacheError, "failed to get sequence length from redis", err)
	}

	return length < threshold, nil
}
