package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf

	ShortUrlDB ShortUrlDB

	Sequence Sequence

	Auth struct {
		AccessSecret string
		AccessExpire int64
	}
}

type ShortUrlDB struct {
	DSN string
}

type Sequence struct {
	DSN string
}
