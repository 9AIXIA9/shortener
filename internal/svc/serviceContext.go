package svc

import (
	"shortener/internal/config"
	"shortener/model"
)

type ServiceContext struct {
	Config      config.Config
	Sequence    model.SequenceModel
	ShortUrlMap model.ShortUrlMapModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
	}
}
