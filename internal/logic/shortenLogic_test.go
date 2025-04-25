package logic

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"shortener/internal/config"
	"shortener/internal/model"
	repositoryMock "shortener/internal/repository/mock"
	"shortener/internal/svc"
	"shortener/internal/types"
	"shortener/pkg/errorx"
	filterMock "shortener/pkg/filter/mock"
	"shortener/pkg/md5"
	urlToolMock "shortener/pkg/urlTool/mock"
	"testing"
)

func TestShortenLogic_Shorten(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建模拟对象
	mockShortUrlMap := repositoryMock.NewMockShortUrlMap(ctrl)
	mockSequence := repositoryMock.NewMockSequence(ctrl)
	mockFilter := filterMock.NewMockFilter(ctrl)
	mockURLClient := urlToolMock.NewMockClient(ctrl)

	// 创建配置
	cfg := config.Config{
		App: config.AppConf{
			Operator:       "test_operator",
			SensitiveWords: []string{"bad"},
			ShortUrlDomain: "example.com",
			ShortUrlPath:   "/short/",
		},
	}

	// 创建ServiceContext
	svcCtx := &svc.ServiceContext{
		Config:                cfg,
		ShortUrlMapRepository: mockShortUrlMap,
		SequenceRepository:    mockSequence,
		Filter:                mockFilter,
	}

	// 测试场景一：无效URL
	t.Run("invalid_url", func(t *testing.T) {
		longURL := "invalid-url"
		mockURLClient.EXPECT().Check(longURL).Return(false, nil)

		l := NewShortenLogic(context.Background(), svcCtx, mockURLClient)
		resp, err := l.Shorten(&types.ShortenRequest{LongUrl: longURL})

		assert.Nil(t, resp)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "failed to connect this URL")
	})

	// 测试场景二：已经是短链接
	t.Run("already_short_url", func(t *testing.T) {
		url := "http://example.com/short/abc123"
		mockURLClient.EXPECT().Check(url).Return(true, nil)

		l := NewShortenLogic(context.Background(), svcCtx, mockURLClient)
		resp, err := l.Shorten(&types.ShortenRequest{LongUrl: url})

		assert.NotNil(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "URL is already shortUrl")
	})

	// 测试场景三：已存在的长链接（通过MD5查询成功）
	t.Run("existing_long_url", func(t *testing.T) {
		longURL := "http://example.com"
		shortURL := "abc123"

		// 动态计算MD5值，与代码中使用相同的算法
		correctMd5, _ := md5.Sum([]byte(longURL))

		// 设置URL检查返回有效
		mockURLClient.EXPECT().Check(longURL).Return(true, nil)

		// 使用正确计算出的MD5值
		mockShortUrlMap.EXPECT().FindOneByMd5(gomock.Any(), correctMd5).Return(&model.ShortUrlMap{
			ShortUrl: shortURL,
		}, nil)

		l := NewShortenLogic(context.Background(), svcCtx, mockURLClient)
		resp, err := l.Shorten(&types.ShortenRequest{LongUrl: longURL})

		assert.Nil(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, svcCtx.Config.App.ShortUrlDomain+svcCtx.Config.App.ShortUrlPath+shortURL, resp.ShortUrl)
	})

	// 测试场景四：新的长链接，需要生成短链接
	t.Run("new_long_url", func(t *testing.T) {
		longURL := "http://newtest.com/page"
		correctMd5, _ := md5.Sum([]byte(longURL))

		mockURLClient.EXPECT().Check(longURL).Return(true, nil)
		mockShortUrlMap.EXPECT().FindOneByMd5(gomock.Any(), correctMd5).Return(nil, errorx.New(errorx.CodeNotFound, "data is not found"))

		// 期望生成序列号并转为短链接
		mockSequence.EXPECT().NextID(gomock.Any()).Return(uint64(12345), nil)

		// 期望存储新的映射
		mockShortUrlMap.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)

		// 期望将短链接添加到过滤器
		mockFilter.EXPECT().AddCtx(gomock.Any(), gomock.Any()).Return(nil)

		l := NewShortenLogic(context.Background(), svcCtx, mockURLClient)
		resp, err := l.Shorten(&types.ShortenRequest{LongUrl: longURL})

		assert.Nil(t, err)
		assert.NotNil(t, resp)
		assert.Contains(t, resp.ShortUrl, "example.com/short/")
	})

	// 测试场景五：生成短链接时遇到敏感词
	t.Run("sensitive_short_url", func(t *testing.T) {
		sensitiveConfig := config.Config{
			App: config.AppConf{
				Operator: "test_operator",
				// 使用包含0-9,a-z,A-Z的敏感词，确保任何生成的短链接都会触发敏感检测
				SensitiveWords: []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b"},
				ShortUrlDomain: "example.com",
				ShortUrlPath:   "/short/",
			},
		}

		sensitiveSvcCtx := &svc.ServiceContext{
			Config:                sensitiveConfig,
			ShortUrlMapRepository: mockShortUrlMap,
			SequenceRepository:    mockSequence,
			Filter:                mockFilter,
		}

		longURL := "http://sensitive.com/page"
		md5Hex, _ := md5.Sum([]byte(longURL))

		mockURLClient.EXPECT().Check(longURL).Return(true, nil)
		mockShortUrlMap.EXPECT().FindOneByMd5(gomock.Any(), md5Hex).Return(nil, errorx.New(errorx.CodeNotFound, "data is not found"))

		// 模拟5次尝试都生成了包含敏感词的短链接
		for i := 0; i < 5; i++ {
			mockSequence.EXPECT().NextID(gomock.Any()).Return(uint64(i+1), nil)
		}

		l := NewShortenLogic(context.Background(), sensitiveSvcCtx, mockURLClient)
		resp, err := l.Shorten(&types.ShortenRequest{LongUrl: longURL})

		assert.Nil(t, resp)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "unable to generate appropriate short link")
	})
}

// 测试URL有效性检查函数
func TestShortenLogic_testConnectivity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockURLClient := urlToolMock.NewMockClient(ctrl)

	t.Run("valid_url", func(t *testing.T) {
		url := "http://valid.com"
		mockURLClient.EXPECT().Check(url).Return(true, nil)

		l := &ShortenLogic{client: mockURLClient}
		result, _ := l.testConnectivity(url)

		assert.True(t, result)
	})

	t.Run("invalid_url", func(t *testing.T) {
		url := "invalid-url"
		mockURLClient.EXPECT().Check(url).Return(false, nil)

		l := &ShortenLogic{client: mockURLClient}
		result, _ := l.testConnectivity(url)

		assert.False(t, result)
	})
}

// 测试短���接检查函数
func TestShortenLogic_inShortUrlDomainPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := config.Config{
		App: config.AppConf{
			Operator:       "test_operator",
			SensitiveWords: []string{"bad"},
			ShortUrlDomain: "example.com",
			ShortUrlPath:   "/short/", // 修改为与业务代码一致的路径
		},
	}

	svcCtx := &svc.ServiceContext{
		Config: cfg,
	}

	t.Run("not_short_url", func(t *testing.T) {
		url := "http://other.com/page"

		l := &ShortenLogic{ctx: context.Background(), svcCtx: svcCtx}
		result := l.inShortUrlDomainPath(url)

		assert.False(t, result) // 不是短链接，返回false
	})

	t.Run("is_short_url", func(t *testing.T) {
		url := "http://example.com/short/abc123"
		l := &ShortenLogic{ctx: context.Background(), svcCtx: svcCtx}
		result := l.inShortUrlDomainPath(url)
		assert.True(t, result)
	})

	// 添加根路径测试
	t.Run("is_root_path", func(t *testing.T) {
		url := "http://example.com/"

		l := &ShortenLogic{ctx: context.Background(), svcCtx: svcCtx}
		result := l.inShortUrlDomainPath(url)

		assert.False(t, result) // 是短链接域名但是根路径，返回 false
	})
}

// 测试MD5转换函数
func TestShortenLogic_convertLongUrlIntoMD5(t *testing.T) {
	l := &ShortenLogic{}

	t.Run("valid_conversion", func(t *testing.T) {
		url := "http://example.com/page"
		m, err := l.convertLongUrlIntoMD5(url)

		assert.Nil(t, err)
		assert.NotEmpty(t, m)
	})
}

// 测试根据MD5查询短链接函数
func TestShortenLogic_findShortUrlByMD5(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockShortUrlMap := repositoryMock.NewMockShortUrlMap(ctrl)

	svcCtx := &svc.ServiceContext{
		ShortUrlMapRepository: mockShortUrlMap,
	}

	t.Run("found", func(t *testing.T) {
		m := "testmd5"
		shortURL := "abc123"
		mockShortUrlMap.EXPECT().FindOneByMd5(gomock.Any(), m).Return(&model.ShortUrlMap{
			ShortUrl: shortURL,
		}, nil)

		l := &ShortenLogic{ctx: context.Background(), svcCtx: svcCtx}
		result, err := l.findShortUrlByMD5(m)

		assert.Nil(t, err)
		assert.Equal(t, shortURL, result)
	})

	t.Run("not_found", func(t *testing.T) {
		m := "nonexistmd5"
		mockShortUrlMap.EXPECT().FindOneByMd5(gomock.Any(), m).Return(nil, errorx.New(errorx.CodeNotFound, "data is not found"))

		l := &ShortenLogic{ctx: context.Background(), svcCtx: svcCtx}
		result, err := l.findShortUrlByMD5(m)

		assert.Empty(t, result)
		assert.Nil(t, err)
	})

	t.Run("repository_error", func(t *testing.T) {
		m := "errormd5"
		mockShortUrlMap.EXPECT().FindOneByMd5(gomock.Any(), m).Return(nil, errors.New("repository error"))

		l := &ShortenLogic{ctx: context.Background(), svcCtx: svcCtx}
		result, err := l.findShortUrlByMD5(m)

		assert.Empty(t, result)
		assert.NotNil(t, err)
	})
}

// 测试短链接生成函数
func TestShortenLogic_generateNonSensitiveShortUrl(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSequence := repositoryMock.NewMockSequence(ctrl)

	// 正确初始化配置
	cfg := config.Config{
		App: config.AppConf{
			SensitiveWords: []string{"bad", "evil"},
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:             cfg,
		SequenceRepository: mockSequence,
	}

	t.Run("success", func(t *testing.T) {
		// 设置正确的模拟调用预期
		mockSequence.EXPECT().NextID(gomock.Any()).Return(uint64(12345), nil)

		l := &ShortenLogic{ctx: context.Background(), svcCtx: svcCtx}
		result, err := l.generateNonSensitiveShortUrl()

		assert.Nil(t, err)
		assert.NotEmpty(t, result)
	})
}

// 测试存储函数
func TestShortenLogic_storeInRepository(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockShortUrlMap := repositoryMock.NewMockShortUrlMap(ctrl)

	cfg := config.Config{
		App: config.AppConf{
			ShortUrlDomain: "example.com",
			Operator:       "test_operator",
		},
	}

	svcCtx := &svc.ServiceContext{
		Config:                cfg,
		ShortUrlMapRepository: mockShortUrlMap,
	}

	t.Run("success", func(t *testing.T) {
		m := "testmd5"
		longURL := "http://example.com/page"
		shortURL := "abc123"

		mockShortUrlMap.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)

		l := &ShortenLogic{ctx: context.Background(), svcCtx: svcCtx}
		err := l.storeInRepository(m, longURL, shortURL)

		assert.Nil(t, err)
	})

	t.Run("repository_error", func(t *testing.T) {
		m := "errormd5"
		longURL := "http://example.com/error"
		shortURL := "error"

		mockShortUrlMap.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(errors.New("insert error"))

		l := &ShortenLogic{ctx: context.Background(), svcCtx: svcCtx}
		err := l.storeInRepository(m, longURL, shortURL)

		assert.NotNil(t, err)
	})
}

// 测试过滤器存储函数
func TestShortenLogic_storeShortUrlInFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFilter := filterMock.NewMockFilter(ctrl)

	svcCtx := &svc.ServiceContext{
		Filter: mockFilter,
	}

	t.Run("success", func(t *testing.T) {
		shortURL := "abc123"
		mockFilter.EXPECT().AddCtx(gomock.Any(), []byte(shortURL)).Return(nil)

		l := &ShortenLogic{ctx: context.Background(), svcCtx: svcCtx}
		err := l.storeShortUrlInFilter(shortURL)

		assert.Nil(t, err)
	})

	t.Run("filter_error", func(t *testing.T) {
		shortURL := "error"
		mockFilter.EXPECT().AddCtx(gomock.Any(), []byte(shortURL)).Return(errors.New("filter error"))

		l := &ShortenLogic{ctx: context.Background(), svcCtx: svcCtx}
		err := l.storeShortUrlInFilter(shortURL)

		assert.NotNil(t, err)
	})
}
