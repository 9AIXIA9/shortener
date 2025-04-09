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
	r, err := redis.NewRedis(redis.RedisConf{
		Host: c.BloomFilterConf.RedisHost,
		Type: c.BloomFilterConf.RedisType,
		Pass: c.BloomFilterConf.RedisPassword,
	})

	if err != nil {
		logx.Severef("NewServiceContext redis.NewRedis failed,err:%v", err)
	}

	return &ServiceContext{
		Config:                c,
		ShortUrlMapRepository: repository.NewShortUrlMap(c.ShortUrlMap.DSN(), c.CacheRedis),
		SequenceRepository:    repository.NewSequence(c.Sequence.DSN()),
		Filter:                filter.NewBloomFilter(r, c.BloomFilterConf.Key, c.BloomFilterConf.Bits),
	}
}
