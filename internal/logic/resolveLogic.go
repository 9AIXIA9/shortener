package logic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"shortener/internal/svc"
	"shortener/internal/types"
	"shortener/internal/types/errorx"
)

type ResolveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewResolveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResolveLogic {
	return &ResolveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ResolveLogic) Resolve(req *types.ResolveRequest) (*types.ResolveResponse, error) {
	//校验参数（handler进行初步处理）

	//进行过滤
	exist, err := l.filter(req.ShortCode)
	if err != nil {
		return nil, err
	}

	if !exist {
		return nil, errorx.New(errorx.CodeNotFound, "the short link does not exist")
	}

	//查询长链
	longUrl, err := l.queryLongUrlByShortUrl(req.ShortCode)
	if err != nil {
		return nil, err
	}

	if len(longUrl) == 0 {
		return nil, errorx.New(errorx.CodeNotFound, "the short link does not exist")
	}

	// 如果数据库中存在，则返回长链接
	return &types.ResolveResponse{OriginalUrl: longUrl}, nil
}

// 查询原始长链接
func (l *ResolveLogic) filter(shortUrl string) (bool, error) {
	exist, err := l.svcCtx.ShortCodeFilter.ExistsCtx(l.ctx, []byte(shortUrl))
	if err != nil {
		return false, errorx.Wrap(err, errorx.CodeSystemError, "fail to check if there is a shortURL through the filter")
	}

	return exist, nil
}

// 查询原始长链接
func (l *ResolveLogic) queryLongUrlByShortUrl(shortUrl string) (string, error) {
	data, err := l.svcCtx.ShortUrlMapRepository.FindOneByShortUrl(l.ctx, shortUrl)
	if err != nil {
		// 对特定错误类型做特殊处理
		if errorx.Is(err, errorx.CodeNotFound) {
			return "", nil
		}
		// 其他错误统一包装
		return "", errorx.Wrap(err, errorx.CodeSystemError, "query short link mapping failed").
			WithContext(l.ctx).
			WithMeta("shortUrl", shortUrl)
	}

	if data == nil {
		return "", nil
	}
	return data.LongUrl, nil
}
