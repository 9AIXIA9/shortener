//go:generate mockgen -source=$GOFILE -destination=./mock/sequence_mock.go -package=repository
package repository

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"shortener/internal/repository/cachex"
	"shortener/internal/repository/database"
	"shortener/pkg/errorx"
	"sync/atomic"
	"time"
)

// Sequence defines the sequence generator interface
type Sequence interface {
	// NextID returns the next unique sequence ID
	NextID(ctx context.Context) (uint64, error)
}

type SequenceOptions struct {
	MaxRetries     int
	RetryBackoff   time.Duration
	ExternPatch    uint64
	CacheThreshold int
	LocalThreshold int
	LocalPatch     uint64
}

func (opt SequenceOptions) WithDefault() SequenceOptions {
	result := opt

	if result.MaxRetries <= 0 {
		result.MaxRetries = 3
	}
	if result.RetryBackoff <= 0 {
		result.RetryBackoff = 50 * time.Millisecond
	}
	if result.ExternPatch <= 0 {
		result.ExternPatch = 1000
	}
	if result.CacheThreshold <= 0 {
		result.CacheThreshold = 20
	}
	if result.LocalThreshold <= 0 {
		result.LocalThreshold = 30
	}
	if result.LocalPatch <= 0 {
		result.LocalPatch = 500
	}

	return result
}

// NewSequence 创建序列生成器
func NewSequence(
	db database.SequenceDatabase,
	externalCache cachex.SequenceCache,
	localCache cachex.SequenceCache,
	opts SequenceOptions,
) Sequence {
	opts = opts.WithDefault()

	seq := &sequence{
		database:       db,
		externalCache:  externalCache,
		localCache:     localCache,
		maxRetries:     opts.MaxRetries,
		retryBackoff:   opts.RetryBackoff,
		externPatch:    opts.ExternPatch,
		cacheThreshold: opts.CacheThreshold,
		localThreshold: opts.LocalThreshold,
		localPatch:     opts.LocalPatch,
	}

	// 检查外部缓存是否可用并设置状态
	if externalCache.IsOK(context.Background()) {
		seq.externalCacheAvailable.Store(true)
		logx.Info("redis cache is healthy")
	} else {
		seq.externalCacheAvailable.Store(false)
		logx.Severef("redis cache is unavailable")
	}

	return seq
}

type sequence struct {
	database      database.SequenceDatabase
	externalCache cachex.SequenceCache
	localCache    cachex.SequenceCache

	externPatch    uint64
	cacheThreshold int
	localPatch     uint64
	localThreshold int

	externalCacheAvailable atomic.Bool
	retryBackoff           time.Duration
	maxRetries             int
}

// NextID generates and returns the next unique ID
func (s *sequence) NextID(ctx context.Context) (uint64, error) {
	// 只有当外部缓存被标记为可用时才尝试从外部缓存获取ID
	if s.externalCacheAvailable.Load() {
		logx.Info("Getting ID from externalCache")
		id, err := s.externalCache.GetSingleID(ctx)
		if err == nil {
			return id, nil
		}

		if errorx.Is(err, errorx.CodeNotFound) {
			ids, err := s.database.GetBatchIDs(ctx, s.externPatch)
			if err == nil {
				if len(ids) == 0 {
					return 0, errorx.New(errorx.CodeNotFound, "database returned empty ID list")
				}

				// 确保使用一致的空切片表示方式
				var remainingIDs []uint64
				if len(ids) > 1 {
					remainingIDs = ids[1:]
				} else {
					remainingIDs = []uint64{} // 明确使用空切片而非nil
				}

				err = s.externalCache.FillIDs(ctx, remainingIDs)
				if err != nil {
					logx.Errorf("external cahce fill ids failed,err:%v,batch first id:%v,batch size:%v", err, ids[0], s.externPatch)
					s.externalCacheAvailable.Store(false)
				}

				return ids[0], nil
			}

			return 0, errorx.Wrap(err, errorx.CodeDatabaseError, "get ids from database failed")
		}

		s.externalCacheAvailable.Store(false)

		logx.Errorf("external cache is unavailable,err:%v,try to fix it", errorx.Wrap(err, errorx.CodeCacheError, "get id from cache failed"))
	}

	//使用本地缓存
	id, err := s.localCache.GetSingleID(ctx)
	if err == nil {
		return id, nil
	}

	if errorx.Is(err, errorx.CodeNotFound) {
		ids, err := s.database.GetBatchIDs(ctx, s.localPatch)
		if err == nil {
			if len(ids) == 0 {
				return 0, errorx.New(errorx.CodeNotFound, "database returned empty ID list")
			}

			// 安全处理缓存填充
			var remainingIDs []uint64
			if len(ids) > 1 {
				remainingIDs = ids[1:]
			}

			err = s.localCache.FillIDs(ctx, remainingIDs)
			if err != nil {
				logx.Errorf("local cache fill ids failed,err:%v", err)
			}

			return ids[0], nil
		}

		return 0, errorx.Wrap(err, errorx.CodeDatabaseError, "get id from database failed")
	}

	logx.Errorf("get id from local cache failed,err:%v", errorx.Wrap(err, errorx.CodeCacheError, "get single id from local cache failed"))

	ids, err := s.database.GetBatchIDs(ctx, 1)
	if err != nil {
		return 0, errorx.Wrap(err, errorx.CodeDatabaseError, "get id from database failed")
	}

	if len(ids) == 0 {
		return 0, errorx.New(errorx.CodeNotFound, "database returned empty ID list")
	}

	return ids[0], nil
}
