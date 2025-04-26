package config

import (
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/rest"
	"time"
)

type Config struct {
	rest.RestConf

	App            AppConf
	ShortUrlMap    ShortUrlConf
	Sequence       SequenceConf
	CacheRedis     cache.CacheConf
	ShortUrlFilter BloomFilterConf
	Auth           AuthConf
	Connect        ConnectConf
}

type AppConf struct {
	Operator       string
	SensitiveWords []string
	ShortUrlDomain string
	ShortUrlPath   string
}

type MysqlConf struct {
	User     string
	Password string
	Host     string
	Port     int
	DBName   string
}

type RedisConf struct {
	Addr     string
	Password string
	Type     string
}

type ShortUrlConf struct {
	Mysql MysqlConf
}

type SequenceConf struct {
	Mysql            MysqlConf
	Redis            RedisConf
	RetryBackoff     time.Duration
	MaxRetries       int
	CachePatch       uint64
	CacheThreshold   int
	LocalPatch       uint64
	LocalThreshold   int
	KeySequenceID    string
	KeySequenceState string
}

type BloomFilterConf struct {
	Redis RedisConf
	Bits  uint
	Key   string
}

type AuthConf struct {
	AccessSecret string
	AccessExpire int64
}

type ConnectConf struct {
	DNSServer       string
	Timeout         time.Duration
	MaxRetries      int
	MaxIdleConns    int
	IdleConnTimeout time.Duration
}

func (db MysqlConf) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&collation=utf8mb4_unicode_ci", db.User, db.Password, db.Host, db.Port, db.DBName)
}
