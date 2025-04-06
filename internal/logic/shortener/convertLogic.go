package shortener

import (
	"context"
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"shortener/internal/model"
	"shortener/internal/svc"
	"shortener/internal/types"
	"shortener/pkg/connect"
	"shortener/pkg/errorx"
	"shortener/pkg/md5"
	"shortener/pkg/urlTool"
)

type ConvertLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewConvertLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConvertLogic {
	return &ConvertLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ConvertLogic) Convert(req *types.ConvertRequest) (resp *types.ConvertResponse, err error) {
	//校验参数
	ok, err := l.checkUrlValid(req.LongUrl)
	if err != nil {
		return nil, errorx.Log(errorx.Error, errorx.CodeInternal,
			"There was an error inside the server",
			logx.Field("url", req.LongUrl), logx.Field("err", err))
	}
	if !ok {
		return nil, errorx.Log(errorx.Info, errorx.CodeLogic,
			"the url entered is invalid",
			logx.Field("url", req.LongUrl))
	}

	//检查此链接是否已有转链
	//计算长链接的MD5
	m, err := md5.Sum([]byte(req.LongUrl))
	if err != nil {
		return nil, errorx.Log(errorx.Error, errorx.CodeInternal,
			"long url sum md5 failed",
			logx.Field("long url", req.LongUrl),
			logx.Field("err", err))
	}

	//数据库查询MD5
	data, err := l.svcCtx.ShortUrlMapModel.FindOneByMd5(l.ctx, m)
	if err != nil && !errors.Is(err, sqlx.ErrNotFound) {
		return nil, errorx.Log(errorx.Error, errorx.CodeInternal,
			"ShortUrlMapModel.FindOneByMd5 failed",
			logx.Field("long url", req.LongUrl),
			logx.Field("err", err))
	}

	if len(data.ShortUrl) != 0 || !errors.Is(err, sqlx.ErrNotFound) {
		return &types.ConvertResponse{ShortUrl: data.ShortUrl}, nil
	}

	//取号
	id, err := l.getID()
	if err != nil {
		return nil, errorx.Log(errorx.Error, errorx.CodeInternal,
			"pick number failed",
			logx.Field("err", err))
	}

	//转链
	shortUrl, err := l.convertUrl(id)
	if err != nil {
		return nil, errorx.Log(errorx.Error, errorx.CodeInternal,
			"convert long links into short link failed",
			logx.Field("long URL", req.LongUrl),
			logx.Field("err", err))
	}

	//存储映射
	_, err = l.svcCtx.ShortUrlMapModel.Insert(l.ctx, &model.ShortUrlMap{
		Id:       id,
		CreateBy: l.svcCtx.Config.Operator,
		IsDel:    0,
		LongUrl:  req.LongUrl,
		Md5:      m,
		ShortUrl: shortUrl,
	})
	if err != nil {
		return nil, errorx.Log(errorx.Error, errorx.CodeInternal,
			"insert url map failed",
			logx.Field("long URL", req.LongUrl),
			logx.Field("err", err))
	}

	//返回响应
	return &types.ConvertResponse{
		ShortUrl: shortUrl,
	}, nil
}

// 检验 Url的合理性
func (l *ConvertLogic) checkUrlValid(URL string) (bool, error) {
	//检查是否可通
	client := connect.NewClient()
	ok, err := connect.Check(client, URL)
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
	data, err := l.svcCtx.ShortUrlMapModel.FindOneByShortUrl(l.ctx, basePath)
	if err != nil && !errors.Is(err, sqlx.ErrNotFound) {
		return false, fmt.Errorf("FindOneByShortUrl failed,err:%w", err)
	}

	if errors.Is(err, sqlx.ErrNotFound) || data == nil {
		return true, nil
	}

	return false, nil
}

// 将长链接转换为MD5
func (l *ConvertLogic) convertMD5(longUrl string) (string, error) {
	return md5.Sum([]byte(longUrl))
}

// 取号
func (l *ConvertLogic) getID() (uint64, error) {
	//todo
	return 0, nil
}

// 转链
func (l *ConvertLogic) convertUrl(id uint64) (string, error) {
	//todo
	return "", nil
}
