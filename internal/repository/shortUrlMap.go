package repository

import (
	"context"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"shortener/internal/model"
)

type ShortUrlMap interface {
	Insert(ctx context.Context, data *model.ShortUrlMap) error
	FindOneByMd5(ctx context.Context, md5 string) (*model.ShortUrlMap, error)
	FindOneByShortUrl(ctx context.Context, shortUrl string) (*model.ShortUrlMap, error)
}

func NewShortUrlMap(dsn string, cacheConf cache.CacheConf) ShortUrlMap {
	conn := sqlx.NewMysql(dsn)
	return shortUrlMap{
		model: model.NewShortUrlMapModel(conn, cacheConf),
	}
}

type shortUrlMap struct {
	model model.ShortUrlMapModel
}

func (s shortUrlMap) Insert(ctx context.Context, data *model.ShortUrlMap) error {
	_, err := s.model.Insert(ctx, data)
	return err
}

func (s shortUrlMap) FindOneByMd5(ctx context.Context, md5 string) (*model.ShortUrlMap, error) {
	return s.model.FindOneByMd5(ctx, md5)
}

func (s shortUrlMap) FindOneByShortUrl(ctx context.Context, shortUrl string) (*model.ShortUrlMap, error) {
	return s.model.FindOneByShortUrl(ctx, shortUrl)
}
