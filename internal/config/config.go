package config

import (
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf

	AppConf

	ShortUrlMap MysqlConf

	Sequence MysqlConf

	CacheRedis cache.CacheConf

	Auth struct {
		AccessSecret string
		AccessExpire int64
	}
}

type AppConf struct {
	Operator       string
	SensitiveWords []string
}

type MysqlConf struct {
	User     string
	Password string
	Host     string
	Port     int
	DBName   string
}

func (db MysqlConf) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&collation=utf8mb4_unicode_ci", db.User, db.Password, db.Host, db.Port, db.DBName)
}
