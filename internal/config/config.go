package config

import (
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/rest"
	"shortener/errorx"
)

type Config struct {
	rest.RestConf

	Operator string

	ShortUrlMap MysqlConf

	Sequence MysqlConf

	CacheRedis cache.CacheConf

	Auth struct {
		AccessSecret string
		AccessExpire int64
	}
}

type MysqlConf struct {
	User     string
	Password string
	Host     string
	Port     int
	DBName   string
}

func (db MysqlConf) Validate() error {
	if db.User == "" {
		return errorx.New(errorx.CodeConfig, "mysql user is empty")
	}
	if db.Password == "" {
		return errorx.New(errorx.CodeConfig, "mysql password is empty")
	}
	if db.Host == "" {
		return errorx.New(errorx.CodeConfig, "mysql host is empty")
	}
	if db.Port == 0 {
		return errorx.New(errorx.CodeConfig, "mysql port is empty")
	}
	if db.DBName == "" {
		return errorx.New(errorx.CodeConfig, "mysql database name is empty")
	}
	return nil
}

func (db MysqlConf) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&collation=utf8mb4_unicode_ci", db.User, db.Password, db.Host, db.Port, db.DBName)
}
