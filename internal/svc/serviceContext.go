package svc

import (
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"shortener/internal/config"
	"shortener/internal/model"
)

type ServiceContext struct {
	Config      config.Config
	Sequence    model.SequenceModel
	ShortUrlMap model.ShortUrlMapModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.ShortUrlMap.DSN())

	model.NewShortUrlMapModel(conn, c.CacheRedis)

	conn = sqlx.NewMysql(c.Sequence.DSN())
	model.NewSequenceModel(conn, c.CacheRedis)

	return &ServiceContext{
		Config: c,
	}
}
