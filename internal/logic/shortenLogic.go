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

	isShortUrl, err := l.isAlreadyShortUrl(req.LongUrl)
	if err != nil {
		return nil, err
	}
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

func (l *ShortenLogic) isAlreadyShortUrl(url string) (bool, error) {
	// 解析URL获取域名和路径
	domain, path := urlTool.GetUrlDomainAndPath(url)

	// 检查是否是我们的短链域名
	if domain == l.svcCtx.Config.App.Domain {
		// 域名匹配，查询数据库验证短链接是否存在
		if path != "" {
			_, err := l.svcCtx.ShortUrlMapRepository.FindOneByShortUrl(l.ctx, path)
			if err != nil {
				if errorx.Is(err, errorx.CodeNotFound) {
					// 在数据库中不存在
					return false, nil
				}
				// 其他数据库错误
				return false, errorx.Wrap(err, errorx.CodeSystemError, "check short url failed")
			}
			// 数据库中存在
			return true, nil
		}
	}

	// 不是短链域名
	return false, nil
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
	return l.svcCtx.Config.App.Domain + "/" + shortUrl
}
