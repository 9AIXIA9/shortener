//go:generate mockgen -source=$GOFILE -destination=./mock/filter_mock.go -package=filter
package filter

import (
	"context"
	"github.com/zeromicro/go-zero/core/bloom"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"shortener/internal/config"
)

type Filter interface {
	AddCtx(ctx context.Context, data []byte) error
	ExistsCtx(ctx context.Context, data []byte) (bool, error)
}

func NewBloomFilter(conf config.BloomFilterConf) Filter {
	// 初始化布隆过滤器Redis连接
	redisConnection, err := redis.NewRedis(redis.RedisConf{
		Host: conf.Redis.Addr,
		Type: conf.Redis.Type,
		Pass: conf.Redis.Password,
	})
	if err != nil {
		logx.Severef("NewServiceContext redis.NewRedis failed,err:%v", err)
	}

	return bloom.New(
		redisConnection,
		conf.Key,
		conf.Bits,
	)
}
