package svc

import (
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
	return &ServiceContext{
		Config:                c,
		ShortUrlMapRepository: repository.NewShortUrlMap(c.ShortUrlMap, c.CacheRedis),
		SequenceRepository:    repository.NewSequence(c.Sequence),
		Filter:                filter.NewBloomFilter(c.ShortUrlFilter),
	}
}
