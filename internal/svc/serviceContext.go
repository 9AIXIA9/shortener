package svc

import (
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"shortener/internal/config"
	model2 "shortener/internal/model"
)

type ServiceContext struct {
	Config      config.Config
	Sequence    model2.SequenceModel
	ShortUrlMap model2.ShortUrlMapModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.ShortUrlDB.DSN)

	model2.NewShortUrlMapModel(conn, c.CacheRedis)

	conn = sqlx.NewMysql(c.SequenceDB.DSN)
	model2.NewSequenceModel(conn, c.CacheRedis)

	return &ServiceContext{
		Config: c,
	}
}
