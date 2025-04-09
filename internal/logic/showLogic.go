package logic

import (
	"context"
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"shortener/pkg/errorx"

	"shortener/internal/svc"
	"shortener/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ShowLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewShowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ShowLogic {
	return &ShowLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ShowLogic) Show(req *types.ShowRequest) (resp *types.ShowResponse, err error) {
	//校验参数（handler进行初步处理）

	//进行过滤
	exist, err := l.filter(req.ShortUrl)
	if err != nil {
		return nil, errorx.Log(errorx.ErrorLevel,
			"showLogic filter failed",
			logx.Field("shortUrl", req.ShortUrl),
			logx.Field("err", err))
	}

	if !exist {
		logx.Debug("the bloom filter think:this url doesn't exist")
		return nil, errorx.Log(errorx.DebugLevel,
			"the requested web page does not exist",
			logx.Field("shortUrl", req.ShortUrl))
	}

	logx.Debug("the bloom filter think:this url exists")

	//查询长链
	longUrl, err := l.queryLongUrlByShortUrl(req.ShortUrl)
	if err != nil {
		return nil, errorx.Log(errorx.ErrorLevel,
			"showLogic queryLongUrlByShortUrl failed",
			logx.Field("shortUrl", req.ShortUrl),
			logx.Field("err", err))
	}

	if len(longUrl) == 0 {
		return nil, errorx.Log(errorx.DebugLevel,
			"the requested web page does not exist",
			logx.Field("shortUrl", req.ShortUrl))
	}

	//返回长链接
	return &types.ShowResponse{LongUrl: longUrl}, nil
}

// 查询原始长链接
func (l *ShowLogic) filter(shortUrl string) (bool, error) {
	exist, err := l.svcCtx.Filter.ExistsCtx(l.ctx, []byte(shortUrl))
	if err != nil {
		return false, fmt.Errorf("svcCtx.Filter.ExistsCtx failed,err:%w", err)
	}

	return exist, nil
}

// 查询原始长链接
func (l *ShowLogic) queryLongUrlByShortUrl(shortUrl string) (string, error) {
	data, err := l.svcCtx.ShortUrlMapRepository.FindOneByShortUrl(l.ctx, shortUrl)
	if err != nil && !errors.Is(err, sqlx.ErrNotFound) {
		return "", fmt.Errorf("FindOneByShortUrl failed,err:%w", err)
	}
	if errors.Is(err, sqlx.ErrNotFound) || data == nil || len(data.LongUrl) == 0 {
		return "", nil
	}
	return data.LongUrl, nil
}
