//go:generate mockgen -source=$GOFILE -destination=./mock/filter_mock.go -package=filter
package filter

import (
	"context"
	"github.com/zeromicro/go-zero/core/bloom"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type Filter interface {
	AddCtx(ctx context.Context, data []byte) error
	ExistsCtx(ctx context.Context, data []byte) (bool, error)
}

func NewBloomFilter(redisConfig *redis.Redis, key string, bits uint) Filter {
	return bloom.New(
		redisConfig,
		key,
		bits,
	)
}
