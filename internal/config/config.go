package config

import (
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf

	App         AppConf
	ShortUrlMap ShortUrlConf
	Sequence    SequenceConf
	CacheRedis  cache.CacheConf
	BloomFilter BloomFilterConf
	Auth        AuthConf
}

type AppConf struct {
	Operator       string
	SensitiveWords []string
	Domain         string
}

type MysqlConf struct {
	User     string
	Password string
	Host     string
	Port     int
	DBName   string
}

type RedisConf struct {
	Host     string
	Password string
	Type     string
}

type ShortUrlConf struct {
	MysqlConf
}

type SequenceConf struct {
	MysqlConf
}

type BloomFilterConf struct {
	Redis RedisConf
	Key   string
	Bits  uint
}

type AuthConf struct {
	AccessSecret string
	AccessExpire int64
}

func (db MysqlConf) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&collation=utf8mb4_unicode_ci", db.User, db.Password, db.Host, db.Port, db.DBName)
}
