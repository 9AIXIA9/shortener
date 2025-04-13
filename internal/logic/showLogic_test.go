package logic

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"shortener/internal/config"
	"shortener/internal/model"
	repositoryMock "shortener/internal/repository/mock"
	"shortener/internal/svc"
	"shortener/internal/types"
	"shortener/pkg/errorx"
	filterMock "shortener/pkg/filter/mock"
	"testing"
)

func TestShowLogic_Show(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建模拟对象
	mockShortUrlMap := repositoryMock.NewMockShortUrlMap(ctrl)
	mockFilter := filterMock.NewMockFilter(ctrl)

	// 创建配置
	cfg := config.Config{}

	// 创建ServiceContext
	svcCtx := &svc.ServiceContext{
		Config:                cfg,
		ShortUrlMapRepository: mockShortUrlMap,
		Filter:                mockFilter,
	}

	// 测试场景一：短链接不存在于过滤器中
	t.Run("short_url_not_in_filter", func(t *testing.T) {
		shortURL := "abc123"

		// 设置过滤器返回不存在
		mockFilter.EXPECT().ExistsCtx(gomock.Any(), []byte(shortURL)).Return(false, nil)

		l := NewShowLogic(context.Background(), svcCtx)
		resp, err := l.Show(&types.ShowRequest{ShortUrl: shortURL})

		assert.Nil(t, resp)
		assert.NotNil(t, err)
		assert.True(t, errorx.Is(err, errorx.CodeNotFound))
	})

	// 测试场景二：过滤器查询出错
	t.Run("filter_error", func(t *testing.T) {
		shortURL := "error"

		// 设置过滤器返回错误
		mockFilter.EXPECT().ExistsCtx(gomock.Any(), []byte(shortURL)).Return(false, errorx.New(errorx.CodeSystemError, "filter error"))

		l := NewShowLogic(context.Background(), svcCtx)
		resp, err := l.Show(&types.ShowRequest{ShortUrl: shortURL})

		assert.Nil(t, resp)
		assert.NotNil(t, err)
		assert.True(t, errorx.Is(err, errorx.CodeSystemError))
	})

	// 测试场景三：短链接存在于过滤器但数据库查询出错
	t.Run("db_query_error", func(t *testing.T) {
		shortURL := "dbError"

		// 设置过滤器返回存在
		mockFilter.EXPECT().ExistsCtx(gomock.Any(), []byte(shortURL)).Return(true, nil)

		// 设置数据库查询返回错误
		mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), shortURL).Return(nil, errorx.New(errorx.CodeSystemError, "database error"))

		l := NewShowLogic(context.Background(), svcCtx)
		resp, err := l.Show(&types.ShowRequest{ShortUrl: shortURL})

		assert.Nil(t, resp)
		assert.NotNil(t, err)
	})

	// 测试场景四：短链接存在且成功返回长链接
	t.Run("success", func(t *testing.T) {
		shortURL := "abc123"
		longURL := "http://example.com/page"

		// 设置过滤器返回存在
		mockFilter.EXPECT().ExistsCtx(gomock.Any(), []byte(shortURL)).Return(true, nil)

		// 设置数据库查询返回成功
		mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), shortURL).Return(&model.ShortUrlMap{
			ShortUrl: shortURL,
			LongUrl:  longURL,
		}, nil)

		l := NewShowLogic(context.Background(), svcCtx)
		resp, err := l.Show(&types.ShowRequest{ShortUrl: shortURL})

		assert.Nil(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, longURL, resp.LongUrl)
	})

	// 测试场景五：短链接在数据库中查询不到
	t.Run("not_found_in_db", func(t *testing.T) {
		shortURL := "notInDb"

		// 设置过滤器返回存在
		mockFilter.EXPECT().ExistsCtx(gomock.Any(), []byte(shortURL)).Return(true, nil)

		// 设置数据库查询返回不存在
		mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), shortURL).Return(nil, errorx.New(errorx.CodeNotFound, "not found"))

		l := NewShowLogic(context.Background(), svcCtx)
		resp, err := l.Show(&types.ShowRequest{ShortUrl: shortURL})

		assert.Nil(t, resp)
		assert.NotNil(t, err)
		assert.True(t, errorx.Is(err, errorx.CodeNotFound))
	})
}

// 测试过滤器检查函数
func TestShowLogic_filter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFilter := filterMock.NewMockFilter(ctrl)

	svcCtx := &svc.ServiceContext{
		Filter: mockFilter,
	}

	t.Run("exists", func(t *testing.T) {
		shortURL := "abc123"
		mockFilter.EXPECT().ExistsCtx(gomock.Any(), []byte(shortURL)).Return(true, nil)

		l := &ShowLogic{ctx: context.Background(), svcCtx: svcCtx}
		exists, err := l.filter(shortURL)

		assert.True(t, exists)
		assert.Nil(t, err)
	})

	t.Run("not_exists", func(t *testing.T) {
		shortURL := "notExists"
		mockFilter.EXPECT().ExistsCtx(gomock.Any(), []byte(shortURL)).Return(false, nil)

		l := &ShowLogic{ctx: context.Background(), svcCtx: svcCtx}
		exists, err := l.filter(shortURL)

		assert.False(t, exists)
		assert.Nil(t, err)
	})

	t.Run("filter_error", func(t *testing.T) {
		shortURL := "error"
		mockFilter.EXPECT().ExistsCtx(gomock.Any(), []byte(shortURL)).Return(false, errorx.New(errorx.CodeSystemError, "filter error"))

		l := &ShowLogic{ctx: context.Background(), svcCtx: svcCtx}
		exists, err := l.filter(shortURL)

		assert.False(t, exists)
		assert.NotNil(t, err)
	})
}

// 测试查询长链接函数
func TestShowLogic_queryLongUrlByShortUrl(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockShortUrlMap := repositoryMock.NewMockShortUrlMap(ctrl)

	svcCtx := &svc.ServiceContext{
		ShortUrlMapRepository: mockShortUrlMap,
	}

	t.Run("found", func(t *testing.T) {
		shortURL := "abc123"
		longURL := "http://example.com/page"
		mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), shortURL).Return(&model.ShortUrlMap{
			ShortUrl: shortURL,
			LongUrl:  longURL,
		}, nil)

		l := &ShowLogic{ctx: context.Background(), svcCtx: svcCtx}
		result, err := l.queryLongUrlByShortUrl(shortURL)

		assert.Equal(t, longURL, result)
		assert.Nil(t, err)
	})

	t.Run("not_found", func(t *testing.T) {
		shortURL := "notFound"
		mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), shortURL).Return(nil, errorx.New(errorx.CodeNotFound, "not found"))

		l := &ShowLogic{ctx: context.Background(), svcCtx: svcCtx}
		result, err := l.queryLongUrlByShortUrl(shortURL)

		assert.Empty(t, result)
		assert.Nil(t, err)
	})

	t.Run("database_error", func(t *testing.T) {
		shortURL := "dbError"
		mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), shortURL).Return(nil, errorx.New(errorx.CodeSystemError, "database error"))

		l := &ShowLogic{ctx: context.Background(), svcCtx: svcCtx}
		result, err := l.queryLongUrlByShortUrl(shortURL)

		assert.Empty(t, result)
		assert.NotNil(t, err)
	})
}
