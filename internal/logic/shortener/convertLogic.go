package shortener

import (
	"context"
	"errors"
	"fmt"
	"github.com/AIXIA/shortener/internal/model"
	"github.com/AIXIA/shortener/internal/svc"
	"github.com/AIXIA/shortener/internal/types"
	"github.com/AIXIA/shortener/pkg/base62"
	"github.com/AIXIA/shortener/pkg/connect"
	"github.com/AIXIA/shortener/pkg/errorx"
	"github.com/AIXIA/shortener/pkg/md5"
	"github.com/AIXIA/shortener/pkg/sensitive"
	"github.com/AIXIA/shortener/pkg/urlTool"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ConvertLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	client connect.Client
}

func NewConvertLogic(ctx context.Context, svcCtx *svc.ServiceContext, client connect.Client) *ConvertLogic {
	return &ConvertLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		client: client,
	}
}

func (l *ConvertLogic) Convert(req *types.ConvertRequest) (resp *types.ConvertResponse, err error) {
	//校验参数
	ok, err := l.checkUrlValid(req.LongUrl)
	if err != nil {
		return nil, errorx.Log(errorx.ErrorLevel, errorx.CodeInternal,
			"convertLogic.checkUrlValid failed",
			logx.Field("url", req.LongUrl), logx.Field("err", err))
	}
	if !ok {
		return nil, errorx.Log(errorx.DebugLevel, errorx.CodeLogic,
			"the url entered is invalid",
			logx.Field("url", req.LongUrl))
	}

	//检查此链接是否已有转链
	//计算长链接的MD5
	m, err := l.convertLongUrlIntoMD5(req.LongUrl)
	if err != nil {
		return nil, errorx.Log(errorx.ErrorLevel, errorx.CodeInternal,
			"long url sum md5 failed",
			logx.Field("long url", req.LongUrl),
			logx.Field("err", err))
	}

	//数据库查询MD5
	data, err := l.findShortUrlByMD5(m)
	if err != nil {
		return nil, errorx.Log(errorx.ErrorLevel, errorx.CodeInternal,
			"ShortUrlMapRepository.FindOneByMd5 failed",
			logx.Field("long url", req.LongUrl),
			logx.Field("err", err))
	}

	if data != nil && len(data.ShortUrl) != 0 {
		shortUrl := l.svcCtx.Config.Domain + "/" + data.ShortUrl
		return &types.ConvertResponse{ShortUrl: shortUrl}, nil
	}

	//转链
	shortUrl, err := l.generateNonSensitiveShortUrl()
	if err != nil {
		return nil, errorx.Log(errorx.ErrorLevel, errorx.CodeInternal,
			"convertLogic.generateNonSensitiveShortUrl failed",
			logx.Field("err", err))
	}

	//存储映射
	err = l.storeInRepository(m, req.LongUrl, shortUrl)
	if err != nil {
		return nil, errorx.Log(errorx.ErrorLevel, errorx.CodeInternal,
			"insert url map failed",
			logx.Field("long URL", req.LongUrl),
			logx.Field("err", err))
	}

	//返回响应
	shortUrl = l.svcCtx.Config.Domain + "/" + shortUrl

	return &types.ConvertResponse{
		ShortUrl: shortUrl,
	}, nil
}

// 检验 Url的合理性
func (l *ConvertLogic) checkUrlValid(URL string) (bool, error) {
	//检查是否可通
	ok, err := l.client.Check(URL)
	if err != nil {
		return false, fmt.Errorf("check connect failed,err:%w", err)
	}

	if !ok {
		return false, nil
	}

	//检查是否已经是短链接
	//获取链接路径
	basePath, err := urlTool.GetBasePath(URL)
	if err != nil {
		return false, fmt.Errorf("get url base path failed,err:%w", err)
	}

	//查询数据库
	data, err := l.svcCtx.ShortUrlMapRepository.FindOneByShortUrl(l.ctx, basePath)
	if err != nil && !errors.Is(err, sqlx.ErrNotFound) {
		return false, fmt.Errorf("FindOneByShortUrl failed,err:%w", err)
	}

	if errors.Is(err, sqlx.ErrNotFound) || data == nil {
		return true, nil
	}

	return false, nil
}

// 将长链接转换为MD5
func (l *ConvertLogic) convertLongUrlIntoMD5(longUrl string) (string, error) {
	return md5.Sum([]byte(longUrl))
}

// 根据md5查询是否已有转链
func (l *ConvertLogic) findShortUrlByMD5(m string) (*model.ShortUrlMap, error) {
	data, err := l.svcCtx.ShortUrlMapRepository.FindOneByMd5(l.ctx, m)
	if errors.Is(err, sqlx.ErrNotFound) {
		return nil, nil
	}
	return data, err
}

// 转化为短链
func (l *ConvertLogic) generateNonSensitiveShortUrl() (string, error) {
	maxAttempts := 5

	for i := 0; i < maxAttempts; i++ {
		id, err := l.svcCtx.SequenceRepository.NextID(l.ctx)
		if err != nil {
			return "", fmt.Errorf("get SequenceRepository.NextID failed: %w", err)
		}
		url := base62.Convert(id)

		// 检查候选链接
		if !sensitive.Exist(l.svcCtx.Config.SensitiveWords, url) {
			return url, nil
		}
		logx.Infof("skipping ID %d, generated short link contains sensitive words: %s", id, url)
	}

	return "", fmt.Errorf("unable to generate appropriate short link after %d batch attempts", maxAttempts)
}

// 数据持久化
func (l *ConvertLogic) storeInRepository(md5 string, longUrl, shortUrl string) error {
	return l.svcCtx.ShortUrlMapRepository.Insert(l.ctx, &model.ShortUrlMap{
		CreateBy: l.svcCtx.Config.Operator,
		IsDel:    0,
		LongUrl:  longUrl,
		Md5:      md5,
		ShortUrl: shortUrl,
	})
}
