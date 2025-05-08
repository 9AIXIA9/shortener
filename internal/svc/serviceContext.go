package svc

import (
	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/rest"
	"shortener/internal/config"
	"shortener/internal/middleware"
	"shortener/internal/repository"
	"shortener/internal/repository/cachex"
	"shortener/internal/repository/database"
	"shortener/internal/types/errorx"
	"shortener/pkg/filter"
	"shortener/pkg/sensitive"
)

const (
	sensitiveWordsPath = "assets/sensitiveWords.txt"
	similarCharsPath   = "assets/similarChars.txt"
	replaceRulesPath   = "assets/replaceRules.txt"
)

type ServiceContext struct {
	Config                config.Config
	SequenceRepository    repository.Sequence
	ShortUrlMapRepository repository.ShortUrlMap
	ShortCodeFilter       filter.Filter
	SensitiveFilter       sensitive.Filter

	Limit rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 创建MySQL连接
	sequenceDB := sqlx.NewMysql(c.Sequence.Mysql.DSN())

	// 创建Redis连接
	sequenceRedis := newRedis(c.Sequence.Redis)

	// 创建数据库访问层
	sequenceDatabase := database.NewMysqlSequenceDatabase(sequenceDB)

	// 创建缓存层
	redisCache := cachex.NewRedisSequenceCache(
		sequenceRedis,
		c.Sequence.KeySequenceID,
		c.Sequence.KeySequenceState,
	)
	localCache := cachex.NewLocalSequenceCache(c.Sequence.LocalCapacity)

	// 创建序列生成器
	sequenceOpts := repository.SequenceOptions{
		MaxRetries:     c.Sequence.MaxRetries,
		RetryBackoff:   c.Sequence.RetryBackoff,
		ExternPatch:    c.Sequence.CachePatch,
		CacheThreshold: c.Sequence.CacheThreshold,
		LocalThreshold: c.Sequence.LocalThreshold,
		LocalPatch:     c.Sequence.LocalPatch,
	}

	// 初始化限流器
	limitRedis := newRedis(c.Limit.Redis)
	tokenLimiter := limit.NewTokenLimiter(c.Limit.Rate, c.Limit.Burst, limitRedis, c.Limit.Key)

	//初始化敏感词过滤器
	f, err := sensitive.NewFilter(sensitiveWordsPath, similarCharsPath, replaceRulesPath)
	if err != nil {
		err = errorx.NewWithCause(errorx.CodeCacheError, "init sensitive words filter failed", err)
		logx.Severef("get sensitive words filter failed,err:%v", err)
	}

	return &ServiceContext{
		Config:                c,
		ShortUrlMapRepository: repository.NewShortUrlMap(c.ShortUrlMap, c.CacheRedis),
		SequenceRepository: repository.NewSequence(
			sequenceDatabase,
			redisCache,
			localCache,
			sequenceOpts,
		),
		ShortCodeFilter: filter.NewBloomFilter(c.ShortUrlFilter),
		SensitiveFilter: f,

		Limit: middleware.NewLimitMiddleware(tokenLimiter).Handle,
	}
}

func newRedis(conf config.RedisConf) *redis.Redis {
	redisConf := redis.RedisConf{
		Host: conf.Addr,
		Type: conf.Type,
		Pass: conf.Password,
	}
	r, err := redis.NewRedis(redisConf)
	if err != nil {
		err = errorx.NewWithCause(errorx.CodeCacheError, "connect to redis fail", err)
		logx.Severef("init redis failed,conf:%v,err:%v", conf, err)
	}
	return r
}
