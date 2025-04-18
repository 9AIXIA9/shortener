package logic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"shortener/internal/model"
	"shortener/internal/svc"
	"shortener/internal/types"
	"shortener/pkg/base62"
	"shortener/pkg/errorx"
	"shortener/pkg/md5"
	"shortener/pkg/sensitive"
	"shortener/pkg/urlTool"
	"strings"
)

type ShortenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	client urlTool.Client
}

func NewShortenLogic(ctx context.Context, svcCtx *svc.ServiceContext, client urlTool.Client) *ShortenLogic {
	return &ShortenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		client: client,
	}
}

func (l *ShortenLogic) Shorten(req *types.ShortenRequest) (*types.ShortenResponse, error) {
	//校验参数
	isValidUrl := l.isValidUrl(req.LongUrl)
	if !isValidUrl {
		return nil, errorx.New(errorx.CodeParamError, "invalid URL")
	}

	isShortUrl := l.inShortUrlDomainPath(req.LongUrl)
	if isShortUrl {
		return nil, errorx.New(errorx.CodeParamError, "URL is already shortUrl")
	}

	//检查此链接是否已有转链
	//计算长链接的MD5
	m, err := l.convertLongUrlIntoMD5(req.LongUrl)
	if err != nil {
		return nil, err
	}

	//数据库查询MD5
	shortUrl, err := l.findShortUrlByMD5(m)
	if errorx.Is(err, errorx.CodeNotFound) || len(shortUrl) == 0 {
		//转链
		shortUrl, err = l.generateNonSensitiveShortUrl()
		if err != nil {
			return nil, err
		}

		//存储映射
		err = l.storeInRepository(m, req.LongUrl, shortUrl)
		if err != nil {
			return nil, err
		}

		//存储过滤
		err = l.storeShortUrlInFilter(shortUrl)
		if err != nil {
			return nil, err
		}

		//返回响应
		return &types.ShortenResponse{
			ShortUrl: l.getFullShortLink(shortUrl),
		}, nil
	}

	if err != nil {
		return nil, err
	}

	if len(shortUrl) != 0 {
		return &types.ShortenResponse{ShortUrl: l.getFullShortLink(shortUrl)}, nil
	}

	return nil, errorx.New(errorx.CodeDatabaseError, "shortUrl is empty")
}

func (l *ShortenLogic) isValidUrl(URL string) bool {
	return l.client.Check(URL)
}

func (l *ShortenLogic) inShortUrlDomainPath(url string) bool {
	domain, path := urlTool.GetUrlDomainAndPath(url)

	if domain == l.svcCtx.Config.App.ShortUrlDomain {
		// 处理配置路径可能带有前导斜杠的情况
		configPath := l.svcCtx.Config.App.ShortUrlPath
		if strings.HasPrefix(configPath, "/") {
			configPath = configPath[1:] // 去掉前导斜杠
		}
		return strings.HasPrefix(path, configPath)
	}

	return false
}

// 将长链接转换为MD5
func (l *ShortenLogic) convertLongUrlIntoMD5(longUrl string) (string, error) {
	m, err := md5.Sum([]byte(longUrl))
	if err != nil {
		return "", errorx.Wrap(err, errorx.CodeSystemError, "fail to convert longUrl into MD5")
	}
	return m, nil
}

// 根据md5查询是否已有转链
func (l *ShortenLogic) findShortUrlByMD5(m string) (string, error) {
	data, err := l.svcCtx.ShortUrlMapRepository.FindOneByMd5(l.ctx, m)
	if err == nil {
		return data.ShortUrl, nil
	}

	if errorx.Is(err, errorx.CodeNotFound) {
		return "", nil
	}

	return "", errorx.Wrap(err, errorx.CodeDatabaseError, "fail to find shortUrlMap by MD5")
}

// 转化为短链
func (l *ShortenLogic) generateNonSensitiveShortUrl() (string, error) {
	maxAttempts := 5

	for i := 0; i < maxAttempts; i++ {
		//获取序号ID
		id, err := l.svcCtx.SequenceRepository.NextID(l.ctx)
		if err != nil {
			return "", errorx.Wrap(err, errorx.CodeDatabaseError, "fail to get sequence next ID")
		}

		//ID转链
		url := base62.Convert(id)

		// 检查敏感词
		if !sensitive.Exist(l.svcCtx.Config.App.SensitiveWords, url) {
			return url, nil
		}
		logx.Infof("skipping ID %d, generated short link contains sensitive words: %s", id, url)
	}

	return "", errorx.New(errorx.CodeServiceUnavailable, "unable to generate appropriate short link after 5 batch attempts").
		WithMeta("maxAttempts", maxAttempts)
}

// 数据持久化
func (l *ShortenLogic) storeInRepository(md5 string, longUrl, shortUrl string) error {
	//存储到仓库中
	err := l.svcCtx.ShortUrlMapRepository.Insert(l.ctx, &model.ShortUrlMap{
		CreateBy: l.svcCtx.Config.App.Operator,
		IsDel:    0,
		LongUrl:  longUrl,
		Md5:      md5,
		ShortUrl: shortUrl,
	})

	if err != nil {
		return errorx.Wrap(err, errorx.CodeDatabaseError, "fail to insert shortUrlMap")
	}
	return nil
}

// 添加到过滤器中
func (l *ShortenLogic) storeShortUrlInFilter(shortUrl string) error {
	err := l.svcCtx.Filter.AddCtx(l.ctx, []byte(shortUrl))
	if err != nil {
		return errorx.Wrap(err, errorx.CodeSystemError, "fail to store shortUrl in filter")
	}
	return nil
}

func (l *ShortenLogic) getFullShortLink(shortUrl string) string {
	return l.svcCtx.Config.App.ShortUrlDomain + l.svcCtx.Config.App.ShortUrlPath + shortUrl
}
