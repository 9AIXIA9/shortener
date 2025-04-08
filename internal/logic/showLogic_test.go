package logic

import (
	"context"
	"errors"
	"testing"

	"shortener/internal/config"
	"shortener/internal/model"
	mockRepository "shortener/internal/repository/mock"
	"shortener/internal/svc"
	"shortener/internal/types"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func TestShowLogic_Show(t *testing.T) {
	testCases := []struct {
		name         string
		req          *types.ShowRequest
		mockSetup    func(mockCtrl *gomock.Controller) *mockRepository.MockShortUrlMap
		expectedResp *types.ShowResponse
		expectedErr  bool
	}{
		{
			name: "成功-找到短链接对应的长链接",
			req: &types.ShowRequest{
				ShortUrl: "abc123",
			},
			mockSetup: func(mockCtrl *gomock.Controller) *mockRepository.MockShortUrlMap {
				mockShortUrlMap := mockRepository.NewMockShortUrlMap(mockCtrl)

				// 模拟数据库查询返回结果
				mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), "abc123").Return(&model.ShortUrlMap{
					LongUrl: "https://example.com/long/path",
				}, nil)

				return mockShortUrlMap
			},
			expectedResp: &types.ShowResponse{
				LongUrl: "https://example.com/long/path",
			},
			expectedErr: false,
		},
		{
			name: "未找到短链接",
			req: &types.ShowRequest{
				ShortUrl: "notfound",
			},
			mockSetup: func(mockCtrl *gomock.Controller) *mockRepository.MockShortUrlMap {
				mockShortUrlMap := mockRepository.NewMockShortUrlMap(mockCtrl)

				// 模拟未找到记录
				mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), "notfound").Return(nil, sqlx.ErrNotFound)

				return mockShortUrlMap
			},
			expectedErr: true,
		},
		{
			name: "数据库查询错误",
			req: &types.ShowRequest{
				ShortUrl: "error",
			},
			mockSetup: func(mockCtrl *gomock.Controller) *mockRepository.MockShortUrlMap {
				mockShortUrlMap := mockRepository.NewMockShortUrlMap(mockCtrl)

				// 模拟数据库查询错误
				mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), "error").Return(nil, errors.New("数据库错误"))

				return mockShortUrlMap
			},
			expectedErr: true,
		},
		{
			name: "找到记录但长链接为空",
			req: &types.ShowRequest{
				ShortUrl: "emptyurl",
			},
			mockSetup: func(mockCtrl *gomock.Controller) *mockRepository.MockShortUrlMap {
				mockShortUrlMap := mockRepository.NewMockShortUrlMap(mockCtrl)

				// 模拟返回空长链接
				mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), "emptyurl").Return(&model.ShortUrlMap{
					LongUrl: "",
				}, nil)

				return mockShortUrlMap
			},
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建mock控制器
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			// 设置mock
			mockShortUrlMap := tc.mockSetup(mockCtrl)

			// 创建服务上下文
			svcCtx := &svc.ServiceContext{
				Config: config.Config{
					AppConf: config.AppConf{
						Domain:         "https://short.com",
						Operator:       "tester",
						SensitiveWords: []string{},
					},
				},
				ShortUrlMapRepository: mockShortUrlMap,
			}

			// 创建测试对象
			logic := NewShowLogic(context.Background(), svcCtx)

			// 执行测试
			resp, err := logic.Show(tc.req)

			// 验证结果
			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedResp, resp)
			}
		})
	}
}
