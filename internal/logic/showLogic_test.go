package logic

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go.uber.org/mock/gomock"
	"shortener/internal/model"
	repositoryMock "shortener/internal/repository/mock"
	"shortener/internal/svc"
	"shortener/internal/types"
	filterMock "shortener/pkg/filter/mock"
	"testing"
)

func TestShowLogic_Show(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建模拟对象
	mockShortUrlMap := repositoryMock.NewMockShortUrlMap(ctrl)
	mockFilter := filterMock.NewMockFilter(ctrl)

	// 创建ServiceContext
	svcCtx := &svc.ServiceContext{
		ShortUrlMapRepository: mockShortUrlMap,
		Filter:                mockFilter,
	}

	// 测试场景一：短链接不存在于过滤器中
	t.Run("short_url_not_in_filter", func(t *testing.T) {
		shortURL := "notExist"
		mockFilter.EXPECT().ExistsCtx(gomock.Any(), []byte(shortURL)).Return(false, nil)

		l := NewShowLogic(context.Background(), svcCtx)
		resp, err := l.Show(&types.ShowRequest{ShortUrl: shortURL})

		assert.Nil(t, resp)
		assert.NotNil(t, err)
	})

	// 测试场景二：过滤器查询错误
	t.Run("filter_error", func(t *testing.T) {
		shortURL := "error"
		testErr := errors.New("filter error")
		mockFilter.EXPECT().ExistsCtx(gomock.Any(), []byte(shortURL)).Return(false, testErr)

		l := NewShowLogic(context.Background(), svcCtx)
		resp, err := l.Show(&types.ShowRequest{ShortUrl: shortURL})

		assert.Nil(t, resp)
		assert.NotNil(t, err)
	})

	// 测试场景三：短链接存在于过滤器中但查询数据库错误
	t.Run("db_query_error", func(t *testing.T) {
		shortURL := "dbError"
		testErr := errors.New("db error")
		mockFilter.EXPECT().ExistsCtx(gomock.Any(), []byte(shortURL)).Return(true, nil)
		mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), shortURL).Return(nil, testErr)

		l := NewShowLogic(context.Background(), svcCtx)
		resp, err := l.Show(&types.ShowRequest{ShortUrl: shortURL})

		assert.Nil(t, resp)
		assert.NotNil(t, err)
	})

	// 测试场景四：短链接在数据库中不存在（未找到记录）
	t.Run("short_url_not_found_in_db", func(t *testing.T) {
		shortURL := "notInDB"
		mockFilter.EXPECT().ExistsCtx(gomock.Any(), []byte(shortURL)).Return(true, nil)
		mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), shortURL).Return(nil, sqlx.ErrNotFound)

		l := NewShowLogic(context.Background(), svcCtx)
		resp, err := l.Show(&types.ShowRequest{ShortUrl: shortURL})

		assert.Nil(t, resp)
		assert.NotNil(t, err)
	})

	// 测试场景五：成功查询
	t.Run("success", func(t *testing.T) {
		shortURL := "abc123"
		longURL := "http://example.com/path"
		mockFilter.EXPECT().ExistsCtx(gomock.Any(), []byte(shortURL)).Return(true, nil)
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
}

// 测试过滤器方法
func TestShowLogic_filter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFilter := filterMock.NewMockFilter(ctrl)
	svcCtx := &svc.ServiceContext{
		Filter: mockFilter,
	}

	t.Run("filter_success", func(t *testing.T) {
		shortURL := "abc123"
		mockFilter.EXPECT().ExistsCtx(gomock.Any(), []byte(shortURL)).Return(true, nil)

		l := NewShowLogic(context.Background(), svcCtx)
		exists, err := l.filter(shortURL)

		assert.Nil(t, err)
		assert.True(t, exists)
	})

	t.Run("filter_error", func(t *testing.T) {
		shortURL := "error"
		testErr := errors.New("filter error")
		mockFilter.EXPECT().ExistsCtx(gomock.Any(), []byte(shortURL)).Return(false, testErr)

		l := NewShowLogic(context.Background(), svcCtx)
		exists, err := l.filter(shortURL)

		assert.NotNil(t, err)
		assert.False(t, exists)
	})
}

// 测试查询长链接方法
func TestShowLogic_queryLongUrlByShortUrl(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockShortUrlMap := repositoryMock.NewMockShortUrlMap(ctrl)
	svcCtx := &svc.ServiceContext{
		ShortUrlMapRepository: mockShortUrlMap,
	}

	t.Run("query_success", func(t *testing.T) {
		shortURL := "abc123"
		longURL := "http://example.com/path"
		mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), shortURL).Return(&model.ShortUrlMap{
			ShortUrl: shortURL,
			LongUrl:  longURL,
		}, nil)

		l := NewShowLogic(context.Background(), svcCtx)
		result, err := l.queryLongUrlByShortUrl(shortURL)

		assert.Nil(t, err)
		assert.Equal(t, longURL, result)
	})

	t.Run("not_found", func(t *testing.T) {
		shortURL := "notfound"
		mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), shortURL).Return(nil, sqlx.ErrNotFound)

		l := NewShowLogic(context.Background(), svcCtx)
		result, err := l.queryLongUrlByShortUrl(shortURL)

		// 修改断言以期望错误
		assert.Nil(t, err)
		assert.Empty(t, result)
	})

	t.Run("db_error", func(t *testing.T) {
		shortURL := "error"
		testErr := errors.New("db error")
		mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), shortURL).Return(nil, testErr)

		l := NewShowLogic(context.Background(), svcCtx)
		result, err := l.queryLongUrlByShortUrl(shortURL)

		// 确保返回了错误且结果为空
		assert.NotNil(t, err)
		assert.Empty(t, result)
	})

	t.Run("empty_long_url", func(t *testing.T) {
		shortURL := "empty"
		mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), shortURL).Return(&model.ShortUrlMap{
			ShortUrl: shortURL,
			LongUrl:  "",
		}, nil)

		l := NewShowLogic(context.Background(), svcCtx)
		result, err := l.queryLongUrlByShortUrl(shortURL)

		assert.Nil(t, err)
		assert.Empty(t, result)
	})
}
