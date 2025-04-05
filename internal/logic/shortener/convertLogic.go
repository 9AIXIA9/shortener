package shortener

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"shortener/internal/model"
	"shortener/internal/svc"
	"shortener/internal/types"
	"shortener/pkg/connect"
	"shortener/pkg/errorx"
	"shortener/pkg/md5"
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
	ok := l.checkUrlValid(req.LongUrl)
	if !ok {
		return nil, errorx.Log(errorx.Info, errorx.CodeLogic, "the url entered is invalid",
			logx.Field("url", req.LongUrl))
	}

	//生成长链接的MD5值
	m, err := l.convertMD5(req.LongUrl)
	if err != nil {
		return nil, errorx.Log(errorx.Error, errorx.CodeInternal, "convert long links into MD5 failed",
			logx.Field("long URL", req.LongUrl), logx.Field("err", err))
	}

	//检查此链接是否已有转链
	data, err := l.svcCtx.ShortUrlMapModel.FindOneByMd5(l.ctx, m)
	if err != nil && !errors.Is(err, sqlx.ErrNotFound) {
		return nil, errorx.Log(errorx.Error, errorx.CodeInternal, "check if this URL exists failed in the database",
			logx.Field("long URL", req.LongUrl),
			logx.Field("err", err))
	}

	//已存在现有短链接
	if len(data.ShortUrl) != 0 || !errors.Is(err, sqlx.ErrNotFound) {
		return &types.ConvertResponse{
			ShortUrl: data.ShortUrl,
		}, nil
	}

	//取号
	id, err := l.getID()
	if err != nil {
		return nil, errorx.Log(errorx.Error, errorx.CodeInternal, "pick number failed",
			logx.Field("err", err))
	}

	//转链
	shortUrl, err := l.convertUrl(id)
	if err != nil {
		return nil, errorx.Log(errorx.Error, errorx.CodeInternal, "convert long links into short link failed",
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
		return nil, errorx.Log(errorx.Error, errorx.CodeInternal, "insert url map failed",
			logx.Field("long URL", req.LongUrl),
			logx.Field("err", err))
	}

	//返回响应
	return &types.ConvertResponse{
		ShortUrl: shortUrl,
	}, nil
}

// 检验 Url的合理性
func (l *ConvertLogic) checkUrlValid(url string) bool {
	//检查是否可通
	client := connect.NewClient()
	if !connect.Check(client, url) {
		return false
	}

	//检查是否为短链接

	return true
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
