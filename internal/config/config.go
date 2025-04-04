package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf

	Operator string

	ShortUrlDB ShortUrlDB

	SequenceDB SequenceDB

	CacheRedis cache.CacheConf

	Auth struct {
		AccessSecret string
		AccessExpire int64
	}
}

type ShortUrlDB struct {
	DSN string
}

type SequenceDB struct {
	DSN string
}
