package svc

import (
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"shortener/internal/config"
	"shortener/internal/model"
)

type ServiceContext struct {
	Config           config.Config
	SequenceModel    model.SequenceModel
	ShortUrlMapModel model.ShortUrlMapModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.ShortUrlMap.DSN())
	conn2 := sqlx.NewMysql(c.Sequence.DSN())

	return &ServiceContext{
		Config:           c,
		SequenceModel:    model.NewSequenceModel(conn, c.CacheRedis),
		ShortUrlMapModel: model.NewShortUrlMapModel(conn2, c.CacheRedis),
	}
}
