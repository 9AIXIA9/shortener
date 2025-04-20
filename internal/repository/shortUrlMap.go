//go:generate mockgen -source=$GOFILE -destination=./mock/shortUrlMap_mock.go -package=repository
package repository

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"shortener/internal/config"
	"shortener/internal/model"
	"shortener/pkg/errorx"
)

// ShortUrlMap 定义短URL映射接口
type ShortUrlMap interface {
	// Insert 添加一个新的URL映射
	Insert(ctx context.Context, data *model.ShortUrlMap) error
	// FindOneByMd5 根据MD5哈希查找URL映射
	FindOneByMd5(ctx context.Context, md5 string) (*model.ShortUrlMap, error)
	// FindOneByShortUrl 根据shortURL查找映射
	FindOneByShortUrl(ctx context.Context, shortUrl string) (*model.ShortUrlMap, error)
}

// NewShortUrlMap 创建短URL映射仓库的新实例
func NewShortUrlMap(conf config.ShortUrlConf, cacheConf cache.CacheConf) ShortUrlMap {
	conn := sqlx.NewMysql(conf.Mysql.DSN())
	return &shortUrlMap{
		model: model.NewShortUrlMapModel(conn, cacheConf),
	}
}

type shortUrlMap struct {
	model model.ShortUrlMapModel
}

// Insert 实现添加URL映射的功能
func (s *shortUrlMap) Insert(ctx context.Context, data *model.ShortUrlMap) error {
	_, err := s.model.Insert(ctx, data)
	if err != nil {
		return errorx.NewWithCause(errorx.CodeDatabaseError, "insert shortUrlMap failed", err).
			WithContext(ctx).WithMeta("data", data)
	}
	return nil
}

// FindOneByMd5 实现通过MD5查找URL映射的功能
func (s *shortUrlMap) FindOneByMd5(ctx context.Context, md5 string) (*model.ShortUrlMap, error) {
	data, err := s.model.FindOneByMd5(ctx, md5)
	return s.handleFindResult(ctx, data, err, "find shortUrlMap by md5 failed")
}

// FindOneByShortUrl 实现通过短URL查找映射的功能
func (s *shortUrlMap) FindOneByShortUrl(ctx context.Context, shortUrl string) (*model.ShortUrlMap, error) {
	data, err := s.model.FindOneByShortUrl(ctx, shortUrl)
	return s.handleFindResult(ctx, data, err, "find shortUrlMap by shortUrl failed")
}

// handleFindResult 处理查询结果和错误
func (s *shortUrlMap) handleFindResult(
	ctx context.Context,
	data *model.ShortUrlMap,
	err error,
	errMsg string,
) (*model.ShortUrlMap, error) {
	if err == nil {
		return data, nil
	}

	if errors.Is(err, sqlx.ErrNotFound) {
		return nil, errorx.Wrap(err, errorx.CodeNotFound, "the data does not exist")
	}

	return nil, errorx.NewWithCause(errorx.CodeDatabaseError, errMsg, err).
		WithContext(ctx)
}
