package svc

import (
	"github.com/AIXIA/shortener/internal/config"
	"github.com/AIXIA/shortener/internal/repository"
)

type ServiceContext struct {
	Config                config.Config
	SequenceRepository    repository.Sequence
	ShortUrlMapRepository repository.ShortUrlMap
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:                c,
		ShortUrlMapRepository: repository.NewShortUrlMap(c.ShortUrlMap.DSN(), c.CacheRedis),
		SequenceRepository:    repository.NewSequence(c.Sequence.DSN()),
	}
}
