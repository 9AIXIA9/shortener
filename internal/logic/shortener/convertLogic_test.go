package shortener

import (
	"context"
	"errors"
	"testing"

	"github.com/AIXIA/shortener/internal/config"
	"github.com/AIXIA/shortener/internal/model"
	mockRepository "github.com/AIXIA/shortener/internal/repository/mock"
	"github.com/AIXIA/shortener/internal/svc"
	"github.com/AIXIA/shortener/internal/types"
	mockConnect "github.com/AIXIA/shortener/pkg/connect/mock"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func TestConvertLogic_Convert(t *testing.T) {
	testCases := []struct {
		name         string
		req          *types.ConvertRequest
		mockSetup    func(mockCtrl *gomock.Controller) (*mockConnect.MockClient, *mockRepository.MockShortUrlMap, *mockRepository.MockSequence)
		expectedResp *types.ConvertResponse
		expectedErr  bool
	}{
		{
			name: "成功-创建新短URL",
			req: &types.ConvertRequest{
				LongUrl: "https://example.com/long/path",
			},
			mockSetup: func(mockCtrl *gomock.Controller) (*mockConnect.MockClient, *mockRepository.MockShortUrlMap, *mockRepository.MockSequence) {
				mockClient := mockConnect.NewMockClient(mockCtrl)
				mockShortUrlMap := mockRepository.NewMockShortUrlMap(mockCtrl)
				mockSequence := mockRepository.NewMockSequence(mockCtrl)

				// URL检查成功
				mockClient.EXPECT().Check("https://example.com/long/path").Return(true, nil)

				// 检查是否已是短链接
				mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), gomock.Any()).Return(nil, sqlx.ErrNotFound)

				// MD5查找结果为空
				mockShortUrlMap.EXPECT().FindOneByMd5(gomock.Any(), gomock.Any()).Return(nil, sqlx.ErrNotFound)

				// 生成ID
				mockSequence.EXPECT().NextID(gomock.Any()).Return(uint64(123456), nil)

				// 插入数据
				mockShortUrlMap.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)

				return mockClient, mockShortUrlMap, mockSequence
			},
			expectedResp: &types.ConvertResponse{
				ShortUrl: "https://short.com/w7e",
			},
			expectedErr: false,
		},
		{
			name: "URL已存在",
			req: &types.ConvertRequest{
				LongUrl: "https://example.com/exists",
			},
			mockSetup: func(mockCtrl *gomock.Controller) (*mockConnect.MockClient, *mockRepository.MockShortUrlMap, *mockRepository.MockSequence) {
				mockClient := mockConnect.NewMockClient(mockCtrl)
				mockShortUrlMap := mockRepository.NewMockShortUrlMap(mockCtrl)
				mockSequence := mockRepository.NewMockSequence(mockCtrl)

				// URL检查成功
				mockClient.EXPECT().Check("https://example.com/exists").Return(true, nil)

				// 检查是否已是短链接
				mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), gomock.Any()).Return(nil, sqlx.ErrNotFound)

				// 已有记录
				mockShortUrlMap.EXPECT().FindOneByMd5(gomock.Any(), gomock.Any()).Return(&model.ShortUrlMap{
					ShortUrl: "existingShort",
				}, nil)

				return mockClient, mockShortUrlMap, mockSequence
			},
			expectedResp: &types.ConvertResponse{
				ShortUrl: "https://short.com/existingShort",
			},
			expectedErr: false,
		},
		{
			name: "URL无效",
			req: &types.ConvertRequest{
				LongUrl: "https://invalid-url.com",
			},
			mockSetup: func(mockCtrl *gomock.Controller) (*mockConnect.MockClient, *mockRepository.MockShortUrlMap, *mockRepository.MockSequence) {
				mockClient := mockConnect.NewMockClient(mockCtrl)
				mockShortUrlMap := mockRepository.NewMockShortUrlMap(mockCtrl)
				mockSequence := mockRepository.NewMockSequence(mockCtrl)

				// URL检查失败
				mockClient.EXPECT().Check("https://invalid-url.com").Return(false, nil)

				return mockClient, mockShortUrlMap, mockSequence
			},
			expectedErr: true,
		},
		{
			name: "URL检查错误",
			req: &types.ConvertRequest{
				LongUrl: "https://example.com/error",
			},
			mockSetup: func(mockCtrl *gomock.Controller) (*mockConnect.MockClient, *mockRepository.MockShortUrlMap, *mockRepository.MockSequence) {
				mockClient := mockConnect.NewMockClient(mockCtrl)
				mockShortUrlMap := mockRepository.NewMockShortUrlMap(mockCtrl)
				mockSequence := mockRepository.NewMockSequence(mockCtrl)

				// URL检查出错
				mockClient.EXPECT().Check("https://example.com/error").Return(false, errors.New("连接错误"))

				return mockClient, mockShortUrlMap, mockSequence
			},
			expectedErr: true,
		},
		{
			name: "数据库MD5查询错误",
			req: &types.ConvertRequest{
				LongUrl: "https://example.com/db-error",
			},
			mockSetup: func(mockCtrl *gomock.Controller) (*mockConnect.MockClient, *mockRepository.MockShortUrlMap, *mockRepository.MockSequence) {
				mockClient := mockConnect.NewMockClient(mockCtrl)
				mockShortUrlMap := mockRepository.NewMockShortUrlMap(mockCtrl)
				mockSequence := mockRepository.NewMockSequence(mockCtrl)

				// URL检查成功
				mockClient.EXPECT().Check("https://example.com/db-error").Return(true, nil)

				// 检查是否已是短链接
				mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), gomock.Any()).Return(nil, sqlx.ErrNotFound)

				// 数据库查询错误
				mockShortUrlMap.EXPECT().FindOneByMd5(gomock.Any(), gomock.Any()).Return(nil, errors.New("数据库错误"))

				return mockClient, mockShortUrlMap, mockSequence
			},
			expectedErr: true,
		},
		{
			name: "序列生成错误",
			req: &types.ConvertRequest{
				LongUrl: "https://example.com/seq-error",
			},
			mockSetup: func(mockCtrl *gomock.Controller) (*mockConnect.MockClient, *mockRepository.MockShortUrlMap, *mockRepository.MockSequence) {
				mockClient := mockConnect.NewMockClient(mockCtrl)
				mockShortUrlMap := mockRepository.NewMockShortUrlMap(mockCtrl)
				mockSequence := mockRepository.NewMockSequence(mockCtrl)

				// URL检查成功
				mockClient.EXPECT().Check("https://example.com/seq-error").Return(true, nil)

				// 检查是否已是短链接
				mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), gomock.Any()).Return(nil, sqlx.ErrNotFound)

				// 没有找到现有记录
				mockShortUrlMap.EXPECT().FindOneByMd5(gomock.Any(), gomock.Any()).Return(nil, sqlx.ErrNotFound)

				// 生成ID失败
				mockSequence.EXPECT().NextID(gomock.Any()).Return(uint64(0), errors.New("序列错误"))

				return mockClient, mockShortUrlMap, mockSequence
			},
			expectedErr: true,
		},
		{
			name: "数据插入错误",
			req: &types.ConvertRequest{
				LongUrl: "https://example.com/insert-error",
			},
			mockSetup: func(mockCtrl *gomock.Controller) (*mockConnect.MockClient, *mockRepository.MockShortUrlMap, *mockRepository.MockSequence) {
				mockClient := mockConnect.NewMockClient(mockCtrl)
				mockShortUrlMap := mockRepository.NewMockShortUrlMap(mockCtrl)
				mockSequence := mockRepository.NewMockSequence(mockCtrl)

				// URL检查成功
				mockClient.EXPECT().Check("https://example.com/insert-error").Return(true, nil)

				// 检查是否已是短链接
				mockShortUrlMap.EXPECT().FindOneByShortUrl(gomock.Any(), gomock.Any()).Return(nil, sqlx.ErrNotFound)

				// 没有找到现有记录
				mockShortUrlMap.EXPECT().FindOneByMd5(gomock.Any(), gomock.Any()).Return(nil, sqlx.ErrNotFound)

				// 生成ID成功
				mockSequence.EXPECT().NextID(gomock.Any()).Return(uint64(12345), nil)

				// 插入失败
				mockShortUrlMap.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(errors.New("插入错误"))

				return mockClient, mockShortUrlMap, mockSequence
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
			mockClient, mockShortUrlMap, mockSequence := tc.mockSetup(mockCtrl)

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
				SequenceRepository:    mockSequence,
			}

			// 创建测试对象
			logic := NewConvertLogic(context.Background(), svcCtx, mockClient)

			// 执行测试
			resp, err := logic.Convert(tc.req)

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
