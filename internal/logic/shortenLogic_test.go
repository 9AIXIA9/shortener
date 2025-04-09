package logic

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go.uber.org/mock/gomock"
	"shortener/internal/config"
	"shortener/internal/model"
	repositoryMock "shortener/internal/repository/mock"
	"shortener/internal/svc"
	"shortener/internal/types"
	connectMock "shortener/pkg/connect/mock"
	"shortener/pkg/errorx"
	filterMock "shortener/pkg/filter/mock"
	"shortener/pkg/md5"
	"testing"
)

// 创建完整的自定义ShortenLogic
type testShortenLogic struct {
	ShortenLogic
	mockGenerateShortUrl func() (string, error)
}

func (l *testShortenLogic) generateNonSensitiveShortUrl() (string, error) {
	if l.mockGenerateShortUrl != nil {
		return l.mockGenerateShortUrl()
	}
	return l.ShortenLogic.generateNonSensitiveShortUrl()
}

// 重要：重写Shorten方法，避免调用原始NextID
func (l *testShortenLogic) Shorten(req *types.ShortenRequest) (*types.ShortenResponse, error) {
	//校验参数
	ok, err := l.checkLongUrlValid(req.LongUrl)
	if err != nil {
		return nil, errorx.Log(errorx.ErrorLevel,
			"convertLogic.checkLongUrlValid failed",
			logx.Field("url", req.LongUrl), logx.Field("err", err))
	}
	if !ok {
		return nil, errorx.Log(errorx.DebugLevel,
			"the url entered is invalid",
			logx.Field("url", req.LongUrl))
	}

	//检查此链接是否已有转链
	//计算长链接的MD5
	m, err := l.convertLongUrlIntoMD5(req.LongUrl)
	if err != nil {
		return nil, errorx.Log(errorx.ErrorLevel,
			"long url sum md5 failed",
			logx.Field("long url", req.LongUrl),
			logx.Field("err", err))
	}

	//数据库查询MD5
	data, err := l.findShortUrlByMD5(m)
	if err != nil {
		return nil, errorx.Log(errorx.ErrorLevel,
			"ShortUrlMapRepository.FindOneByMd5 failed",
			logx.Field("long url", req.LongUrl),
			logx.Field("err", err))
	}

	if data != nil && len(data.ShortUrl) != 0 {
		shortUrl := l.svcCtx.Config.Domain + "/" + data.ShortUrl
		return &types.ShortenResponse{ShortUrl: shortUrl}, nil
	}

	//转链 - 使用我们的模拟方法
	shortUrl, err := l.generateNonSensitiveShortUrl()
	if err != nil {
		return nil, errorx.Log(errorx.ErrorLevel,
			"convertLogic.generateNonSensitiveShortUrl failed",
			logx.Field("err", err))
	}

	//存储映射
	err = l.storeInRepository(m, req.LongUrl, shortUrl)
	if err != nil {
		return nil, errorx.Log(errorx.ErrorLevel,
			"convertLogic.storeInRepository failed",
			logx.Field("long URL", req.LongUrl),
			logx.Field("err", err))
	}

	//存储过滤
	err = l.storeShortUrlInFilter(shortUrl)
	if err != nil {
		return nil, errorx.Log(errorx.ErrorLevel,
			"convertLogic.storeShortUrlInFilter failed",
			logx.Field("long URL", req.LongUrl),
			logx.Field("short URL", shortUrl),
			logx.Field("err", err))
	}

	//返回响应
	shortUrl = l.svcCtx.Config.Domain + "/" + shortUrl

	return &types.ShortenResponse{
		ShortUrl: shortUrl,
	}, nil
}

func newTestShortenLogic(ctx context.Context, svcCtx *svc.ServiceContext, client *connectMock.MockClient) *testShortenLogic {
	return &testShortenLogic{
		ShortenLogic: ShortenLogic{
			Logger: logx.WithContext(ctx),
			ctx:    ctx,
			svcCtx: svcCtx,
			client: client,
		},
	}
}

func TestShortenLogic_Shorten(t *testing.T) {
	// 保存原始md5.MockSum函数，便于后续恢复
	originalMd5Sum := md5.MockSum
	defer func() { md5.MockSum = originalMd5Sum }()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建模拟对象
	mockShortUrlMap := repositoryMock.NewMockShortUrlMap(ctrl)
	mockSequence := repositoryMock.NewMockSequence(ctrl)
	mockClient := connectMock.NewMockClient(ctrl)
	mockFilter := filterMock.NewMockFilter(ctrl)

	// 创建ServiceContext
	svcCtx := &svc.ServiceContext{
		ShortUrlMapRepository: mockShortUrlMap,
		SequenceRepository:    mockSequence,
		Filter:                mockFilter,
		Config: config.Config{
			AppConf: config.AppConf{
				Domain:         "http://example.com",
				Operator:       "test",
				SensitiveWords: []string{"bad"},
			},
		},
	}

	// 测试场景一：URL 无效
	t.Run("invalid_url", func(t *testing.T) {
		mockClient.EXPECT().Check("http://invalid.com").Return(false, nil)

		l := NewShortenLogic(context.Background(), svcCtx, mockClient)
		resp, err := l.Shorten(&types.ShortenRequest{LongUrl: "http://invalid.com"})

		assert.Nil(t, resp)
		assert.NotNil(t, err)
	})

	// 测试场景二：URL 已存在对应的短链接
	t.Run("existing_url", func(t *testing.T) {
		longURL := "http://example.org/test"
		md5Value := "abcd1234"
		shortURL := "xyz789"

		// 修改md5.MockSum函数行为
		md5.MockSum = func(data []byte) (string, error) {
			return md5Value, nil
		}

		mockClient.EXPECT().Check(longURL).Return(true, nil)
		mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), gomock.Any()).Return(nil, sqlx.ErrNotFound)
		mockShortUrlMap.EXPECT().FindOneByMd5(gomock.Any(), md5Value).Return(&model.ShortUrlMap{
			ShortUrl: shortURL,
			LongUrl:  longURL,
			Md5:      md5Value,
		}, nil)

		l := NewShortenLogic(context.Background(), svcCtx, mockClient)
		resp, err := l.Shorten(&types.ShortenRequest{LongUrl: longURL})

		assert.Nil(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "http://example.com/"+shortURL, resp.ShortUrl)
	})

	// 测试场景三：创建新的短链接
	t.Run("create_new_short_url", func(t *testing.T) {
		longURL := "http://example.org/new"
		md5Value := "efgh5678"
		shortURL := "abc123"

		// 修改md5.MockSum函数行为
		md5.MockSum = func(data []byte) (string, error) {
			return md5Value, nil
		}

		mockClient.EXPECT().Check(longURL).Return(true, nil)
		mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), gomock.Any()).Return(nil, sqlx.ErrNotFound)
		mockShortUrlMap.EXPECT().FindOneByMd5(gomock.Any(), md5Value).Return(nil, sqlx.ErrNotFound)
		// 这两行必须有预期设置
		mockShortUrlMap.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)
		mockFilter.EXPECT().AddCtx(gomock.Any(), gomock.Any()).Return(nil)

		// 使用自定义的测试逻辑类
		l := newTestShortenLogic(context.Background(), svcCtx, mockClient)
		l.mockGenerateShortUrl = func() (string, error) {
			return shortURL, nil
		}

		resp, err := l.Shorten(&types.ShortenRequest{LongUrl: longURL})

		assert.Nil(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "http://example.com/"+shortURL, resp.ShortUrl)
	})

	// 测试场景四：生成短链接失败
	t.Run("failed_to_generate_short_url", func(t *testing.T) {
		longURL := "http://example.org/fail"
		md5Value := "ijkl9012"
		genError := errors.New("failed to generate short URL")

		// 修改md5.MockSum函数行为
		md5.MockSum = func(data []byte) (string, error) {
			return md5Value, nil
		}

		mockClient.EXPECT().Check(longURL).Return(true, nil)
		mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), gomock.Any()).Return(nil, sqlx.ErrNotFound)
		mockShortUrlMap.EXPECT().FindOneByMd5(gomock.Any(), md5Value).Return(nil, sqlx.ErrNotFound)

		// 使用自定义的测试逻辑类
		l := newTestShortenLogic(context.Background(), svcCtx, mockClient)
		l.mockGenerateShortUrl = func() (string, error) {
			return "", genError
		}

		resp, err := l.Shorten(&types.ShortenRequest{LongUrl: longURL})

		assert.Nil(t, resp)
		assert.NotNil(t, err)
	})
}

func TestShortenLogic_checkLongUrlValid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockShortUrlMap := repositoryMock.NewMockShortUrlMap(ctrl)
	mockClient := connectMock.NewMockClient(ctrl)

	svcCtx := &svc.ServiceContext{
		ShortUrlMapRepository: mockShortUrlMap,
	}

	t.Run("valid_url", func(t *testing.T) {
		mockClient.EXPECT().Check("http://valid.com").Return(true, nil)
		mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), gomock.Any()).Return(nil, sqlx.ErrNotFound)

		l := NewShortenLogic(context.Background(), svcCtx, mockClient)
		ok, err := l.checkLongUrlValid("http://valid.com")

		assert.Nil(t, err)
		assert.True(t, ok)
	})

	t.Run("already_shortened_url", func(t *testing.T) {
		shortUrl := "abc123"
		mockClient.EXPECT().Check(gomock.Any()).Return(true, nil)
		mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), shortUrl).Return(&model.ShortUrlMap{
			ShortUrl: shortUrl,
		}, nil)

		l := NewShortenLogic(context.Background(), svcCtx, mockClient)
		ok, err := l.checkLongUrlValid("http://example.com/" + shortUrl)

		assert.Nil(t, err)
		assert.False(t, ok)
	})
}

func TestShortenLogic_generateNonSensitiveShortUrl(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSequence := repositoryMock.NewMockSequence(ctrl)

	svcCtx := &svc.ServiceContext{
		SequenceRepository: mockSequence,
		Config: config.Config{
			AppConf: config.AppConf{
				SensitiveWords: []string{"bad"},
			},
		},
	}

	t.Run("success_generate", func(t *testing.T) {
		mockSequence.EXPECT().NextID(gomock.Any()).Return(uint64(123456), nil)

		l := NewShortenLogic(context.Background(), svcCtx, nil)
		shortUrl, err := l.generateNonSensitiveShortUrl()

		assert.Nil(t, err)
		assert.NotEmpty(t, shortUrl)
	})
}
