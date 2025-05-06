package svc

import (
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"shortener/internal/config"
	"shortener/internal/repository"
	"shortener/internal/repository/cachex"
	"shortener/internal/repository/database"
	"shortener/pkg/errorx"
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
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 创建MySQL连接
	sequenceDB := sqlx.NewMysql(c.Sequence.Mysql.DSN())

	// 创建Redis连接
	redisConf := redis.RedisConf{
		Host: c.Sequence.Redis.Addr,
		Type: c.Sequence.Redis.Type,
		Pass: c.Sequence.Redis.Password,
	}
	sequenceRedis, err := redis.NewRedis(redisConf)
	if err != nil {
		err = errorx.NewWithCause(errorx.CodeDatabaseError, "connect to redis failed", err)
		logx.Severef("init service context failed,err:%v", err)
	}

	f, err := sensitive.NewFilter(sensitiveWordsPath, similarCharsPath, replaceRulesPath)
	if err != nil {
		logx.Severef("init service context failed,err:%v", err)
	}

	// 创建数据库访问层
	sequenceDatabase := database.NewMysqlSequenceDatabase(sequenceDB)

	// 创建缓存层
	redisCache := cachex.NewRedisSequenceCache(
		sequenceRedis,
		c.Sequence.KeySequenceID,
		c.Sequence.KeySequenceState,
	)
	localCache := cachex.NewLocalSequenceCache()

	// 创建序列生成器
	sequenceOpts := repository.SequenceOptions{
		MaxRetries:     c.Sequence.MaxRetries,
		RetryBackoff:   c.Sequence.RetryBackoff,
		ExternPatch:    c.Sequence.CachePatch,
		CacheThreshold: c.Sequence.CacheThreshold,
		LocalThreshold: c.Sequence.LocalThreshold,
		LocalPatch:     c.Sequence.LocalPatch,
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
	}
}
