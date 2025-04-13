package svc

import (
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"shortener/internal/config"
	"shortener/internal/repository"
	"shortener/pkg/filter"
)

type ServiceContext struct {
	Config                config.Config
	SequenceRepository    repository.Sequence
	ShortUrlMapRepository repository.ShortUrlMap
	Filter                filter.Filter
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化布隆过滤器Redis连接
	r, err := redis.NewRedis(redis.RedisConf{
		Host: c.BloomFilter.Redis.Host,
		Type: c.BloomFilter.Redis.Type,
		Pass: c.BloomFilter.Redis.Password,
	})

	if err != nil {
		logx.Severef("NewServiceContext redis.NewRedis failed,err:%v", err)
	}

	return &ServiceContext{
		Config:                c,
		ShortUrlMapRepository: repository.NewShortUrlMap(c.ShortUrlMap.DSN(), c.CacheRedis),
		SequenceRepository:    repository.NewSequence(c.Sequence.DSN()),
		Filter:                filter.NewBloomFilter(r, c.BloomFilter.Key, c.BloomFilter.Bits),
	}
}
