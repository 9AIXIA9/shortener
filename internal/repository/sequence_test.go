package repository

import (
	"context"
	"errors"
	"shortener/internal/repository/cachex"
	cachexMock "shortener/internal/repository/cachex/mock"
	databaseMock "shortener/internal/repository/database/mock"
	"shortener/internal/types/errorx"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSequence_NextID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建mock对象
	mockDB := databaseMock.NewMockSequenceDatabase(ctrl)
	mockExternalCache := cachexMock.NewMockSequenceCache(ctrl)
	mockLocalCache := cachexMock.NewMockSequenceCache(ctrl)

	// 测试配置
	opts := SequenceOptions{
		MaxRetries:     3,
		RetryBackoff:   50 * time.Millisecond,
		ExternPatch:    1000,
		CacheThreshold: 20,
		LocalThreshold: 30,
		LocalPatch:     500,
	}

	tests := []struct {
		name               string
		setupMocks         func()
		expectedID         uint64
		expectedErr        error
		expectedErrCode    errorx.Code
		externalCacheState bool
	}{
		{
			name: "从外部缓存获取ID成功",
			setupMocks: func() {
				mockExternalCache.EXPECT().GetSingleID(gomock.Any()).Return(uint64(123), nil)
			},
			expectedID:         123,
			expectedErr:        nil,
			externalCacheState: true,
		},
		{
			name: "外部缓存为空，从数据库批量获取并填充缓存",
			setupMocks: func() {
				mockExternalCache.EXPECT().GetSingleID(gomock.Any()).Return(uint64(0), errorx.New(errorx.CodeNotFound, "无可用ID"))
				mockDB.EXPECT().GetBatchIDs(gomock.Any(), uint64(1000)).Return([]uint64{100, 101, 102}, nil)
				mockExternalCache.EXPECT().FillIDs(gomock.Any(), []uint64{101, 102}).Return(nil)
			},
			expectedID:         100,
			expectedErr:        nil,
			externalCacheState: true,
		},
		{
			name: "外部缓存失效，使用本地缓存",
			setupMocks: func() {
				// 预填充本地缓存
				mockLocalCache.EXPECT().GetSingleID(gomock.Any()).Return(uint64(200), nil)
			},
			expectedID:         200,
			expectedErr:        nil,
			externalCacheState: false, // 外部缓存应被标记为不可用
		},
		{
			name: "所有缓存都失败，直接从数据库获取",
			setupMocks: func() {
				// 当外部缓存被标记为不可用时，确保不会调用GetSingleID

				// 本地缓存失败
				mockLocalCache.EXPECT().GetSingleID(gomock.Any()).Return(uint64(0), errorx.New(errorx.CodeSystemError, "本地缓存错误"))
				// 直接从数据库获取ID - 应返回999
				mockDB.EXPECT().GetBatchIDs(gomock.Any(), uint64(1)).Return([]uint64{999}, nil)
			},
			expectedID:         999,
			expectedErr:        nil,
			externalCacheState: false, // 外部缓存不���用
		},
		{
			name: "所有途径获取ID都失败",
			setupMocks: func() {
				// 本地缓存返回错误
				mockLocalCache.EXPECT().GetSingleID(gomock.Any()).Return(uint64(0), errorx.New(errorx.CodeSystemError, "本地缓存错误"))
				// 模拟数据库返回错误
				mockDB.EXPECT().GetBatchIDs(gomock.Any(), uint64(1)).Return(nil, errorx.New(errorx.CodeDatabaseError, "数据库错误"))
			},
			expectedID:         0,
			expectedErr:        errorx.New(errorx.CodeDatabaseError, "数据库错误"),
			expectedErrCode:    errorx.CodeDatabaseError,
			externalCacheState: false, // 外部缓存不可用
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建序列生成器实例
			seq := &sequence{
				database:       mockDB,
				externalCache:  mockExternalCache,
				localCache:     mockLocalCache,
				maxRetries:     opts.MaxRetries,
				retryBackoff:   opts.RetryBackoff,
				externPatch:    opts.ExternPatch,
				cacheThreshold: opts.CacheThreshold,
				localThreshold: opts.LocalThreshold,
				localPatch:     opts.LocalPatch,
			}

			// 预先设置外部缓存状态，这决定了执行路���
			seq.externalCacheAvailable.Store(tt.externalCacheState)

			// 配置mock行为
			tt.setupMocks()

			// 执行测试
			id, err := seq.NextID(context.Background())

			// 验证结果
			if tt.expectedErr == nil {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			} else {
				assert.Error(t, err)
				var targetError *errorx.ErrorX
				if errors.As(err, &targetError) {
					assert.Equal(t, tt.expectedErrCode, targetError.Code)
				}
			}
		})
	}
}

// 测试外部缓存填充失败的情况
func TestSequence_ExternalCacheFillFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := databaseMock.NewMockSequenceDatabase(ctrl)
	mockExternalCache := cachexMock.NewMockSequenceCache(ctrl)
	localCache := cachex.NewLocalSequenceCache()

	seq := &sequence{
		database:       mockDB,
		externalCache:  mockExternalCache,
		localCache:     localCache,
		maxRetries:     3,
		retryBackoff:   50 * time.Millisecond,
		externPatch:    1000,
		cacheThreshold: 20,
		localThreshold: 30,
		localPatch:     500,
	}

	seq.externalCacheAvailable.Store(true)

	mockExternalCache.EXPECT().GetSingleID(gomock.Any()).Return(uint64(0), errorx.New(errorx.CodeNotFound, "缓存为空"))
	mockDB.EXPECT().GetBatchIDs(gomock.Any(), uint64(1000)).Return([]uint64{100, 101, 102}, nil)
	mockExternalCache.EXPECT().FillIDs(gomock.Any(), []uint64{101, 102}).Return(errorx.New(errorx.CodeCacheError, "填充缓存失败"))

	id, err := seq.NextID(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, uint64(100), id)
	assert.False(t, seq.externalCacheAvailable.Load(), "外部缓存应标记为不可用")
}

// 测试本地缓存返回非NotFound错误
func TestSequence_LocalCacheNonNotFoundError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := databaseMock.NewMockSequenceDatabase(ctrl)
	mockExternalCache := cachexMock.NewMockSequenceCache(ctrl)
	mockLocalCache := cachexMock.NewMockSequenceCache(ctrl)

	seq := &sequence{
		database:       mockDB,
		externalCache:  mockExternalCache,
		localCache:     mockLocalCache,
		maxRetries:     3,
		retryBackoff:   50 * time.Millisecond,
		externPatch:    1000,
		cacheThreshold: 20,
		localThreshold: 30,
		localPatch:     500,
	}

	seq.externalCacheAvailable.Store(false)

	// 本地缓存返回系统错误(非NotFound)
	mockLocalCache.EXPECT().GetSingleID(gomock.Any()).Return(uint64(0), errorx.New(errorx.CodeSystemError, "系统错误"))
	mockDB.EXPECT().GetBatchIDs(gomock.Any(), uint64(1)).Return([]uint64{300}, nil)

	id, err := seq.NextID(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, uint64(300), id)
}

// 测试本地缓存填充后的连续获取
func TestSequence_LocalCacheContinuousGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := databaseMock.NewMockSequenceDatabase(ctrl)
	mockExternalCache := cachexMock.NewMockSequenceCache(ctrl)
	localCache := cachex.NewLocalSequenceCache()

	seq := &sequence{
		database:       mockDB,
		externalCache:  mockExternalCache,
		localCache:     localCache,
		maxRetries:     3,
		retryBackoff:   50 * time.Millisecond,
		externPatch:    1000,
		cacheThreshold: 20,
		localThreshold: 30,
		localPatch:     500,
	}

	seq.externalCacheAvailable.Store(false)

	// 首次调用，本地缓存为空
	mockDB.EXPECT().GetBatchIDs(gomock.Any(), uint64(500)).Return([]uint64{200, 201, 202, 203}, nil)

	// 第一次请求
	id1, err1 := seq.NextID(context.Background())
	assert.NoError(t, err1)
	assert.Equal(t, uint64(200), id1)

	// 后续请求应直接从本地缓存获取
	id2, err2 := seq.NextID(context.Background())
	assert.NoError(t, err2)
	assert.Equal(t, uint64(201), id2)

	id3, err3 := seq.NextID(context.Background())
	assert.NoError(t, err3)
	assert.Equal(t, uint64(202), id3)
}

// 测试数据库返回单个ID的边界情况
func TestSequence_SingleIDFromDatabase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := databaseMock.NewMockSequenceDatabase(ctrl)
	mockExternalCache := cachexMock.NewMockSequenceCache(ctrl)
	mockLocalCache := cachexMock.NewMockSequenceCache(ctrl) // 修改为mock对象

	seq := &sequence{
		database:       mockDB,
		externalCache:  mockExternalCache,
		localCache:     mockLocalCache, // 使用mock本地缓存
		maxRetries:     3,
		retryBackoff:   50 * time.Millisecond,
		externPatch:    1000,
		cacheThreshold: 20,
		localThreshold: 30,
		localPatch:     500,
	}

	seq.externalCacheAvailable.Store(true)

	mockExternalCache.EXPECT().GetSingleID(gomock.Any()).Return(uint64(0), errorx.New(errorx.CodeNotFound, "缓存为空"))
	mockDB.EXPECT().GetBatchIDs(gomock.Any(), uint64(1000)).Return([]uint64{400}, nil)
	// 使用gomock.Any()代替明确指定空切片，这样可以匹配任何切片包括nil和空切片
	mockExternalCache.EXPECT().FillIDs(gomock.Any(), gomock.Any()).Return(nil)

	id, err := seq.NextID(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, uint64(400), id)
}

// 测试NewSequence函数
func TestNewSequence(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := databaseMock.NewMockSequenceDatabase(ctrl)
	mockExternalCache := cachexMock.NewMockSequenceCache(ctrl)
	localCache := cachex.NewLocalSequenceCache()

	// 测试默认选项
	t.Run("使用默认选项", func(t *testing.T) {
		mockExternalCache.EXPECT().IsOK(gomock.Any()).Return(true)

		seq := NewSequence(mockDB, mockExternalCache, localCache, SequenceOptions{})

		assert.NotNil(t, seq)
	})

	// 测试缓存不可用
	t.Run("外部缓存不可用", func(t *testing.T) {
		mockExternalCache.EXPECT().IsOK(gomock.Any()).Return(false)

		seq := NewSequence(mockDB, mockExternalCache, localCache, SequenceOptions{})

		assert.NotNil(t, seq)
		// 检验内部状态 - 这需要将sequence.externalCacheAvailable改为导出字段或提供访问方法
		s := seq.(*sequence)
		assert.False(t, s.externalCacheAvailable.Load())
	})
}

// 测试SequenceOptions.WithDefault方法
func TestSequenceOptions_WithDefault(t *testing.T) {
	// 测试空选项
	t.Run("空选项使用默认值", func(t *testing.T) {
		opts := SequenceOptions{}
		result := opts.WithDefault()

		assert.Equal(t, 3, result.MaxRetries)
		assert.Equal(t, 50*time.Millisecond, result.RetryBackoff)
		assert.Equal(t, uint64(1000), result.ExternPatch)
		assert.Equal(t, 20, result.CacheThreshold)
		assert.Equal(t, 30, result.LocalThreshold)
		assert.Equal(t, uint64(500), result.LocalPatch)
	})

	// 测试自定义选项
	t.Run("自定义选项", func(t *testing.T) {
		opts := SequenceOptions{
			MaxRetries:     5,
			RetryBackoff:   100 * time.Millisecond,
			ExternPatch:    2000,
			CacheThreshold: 40,
			LocalThreshold: 60,
			LocalPatch:     1000,
		}
		result := opts.WithDefault()

		assert.Equal(t, 5, result.MaxRetries)
		assert.Equal(t, 100*time.Millisecond, result.RetryBackoff)
		assert.Equal(t, uint64(2000), result.ExternPatch)
		assert.Equal(t, 40, result.CacheThreshold)
		assert.Equal(t, 60, result.LocalThreshold)
		assert.Equal(t, uint64(1000), result.LocalPatch)
	})
}

// 测试外部缓存状态恢复
func TestSequence_ExternalCacheRecovery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := databaseMock.NewMockSequenceDatabase(ctrl)
	mockExternalCache := cachexMock.NewMockSequenceCache(ctrl)
	localCache := cachex.NewLocalSequenceCache()

	seq := &sequence{
		database:       mockDB,
		externalCache:  mockExternalCache,
		localCache:     localCache,
		maxRetries:     3,
		retryBackoff:   50 * time.Millisecond,
		externPatch:    1000,
		cacheThreshold: 20,
		localThreshold: 30,
		localPatch:     500,
	}

	// 初始状态：外部缓存不可用
	seq.externalCacheAvailable.Store(false)

	// 阶段1：外部缓存仍不可用，使用本地缓存
	t.Run("外部缓存不可用时使用本地缓存", func(t *testing.T) {
		err := localCache.FillIDs(context.Background(), []uint64{100})
		assert.NoError(t, err)

		id, err := seq.NextID(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, uint64(100), id)
		assert.False(t, seq.externalCacheAvailable.Load())
	})

	// 阶段2：外部缓存恢复，但本地缓存已用完
	t.Run("外部缓存恢复后使用外部缓存", func(t *testing.T) {
		// 模拟外部缓存恢复
		mockExternalCache.EXPECT().GetSingleID(gomock.Any()).Return(uint64(200), nil).Times(1)

		// 手动调用检查外部缓存是否可用的方法
		// 假设有一个内部方法会定期检查外部缓存状态
		seq.externalCacheAvailable.Store(true)

		id, err := seq.NextID(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, uint64(200), id)
		assert.True(t, seq.externalCacheAvailable.Load())
	})
}

// 测试批量获取ID
func TestSequence_BatchGetIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := databaseMock.NewMockSequenceDatabase(ctrl)
	mockExternalCache := cachexMock.NewMockSequenceCache(ctrl)
	localCache := cachex.NewLocalSequenceCache()

	seq := &sequence{
		database:       mockDB,
		externalCache:  mockExternalCache,
		localCache:     localCache,
		maxRetries:     3,
		retryBackoff:   50 * time.Millisecond,
		externPatch:    10, // 小批量，易于测试
		cacheThreshold: 20,
		localThreshold: 30,
		localPatch:     5, // 小批量，易于测试
	}

	// 启用外部缓存
	seq.externalCacheAvailable.Store(true)

	// 批量获取ID并填充缓存
	t.Run("首次批量获取ID并填充缓存", func(t *testing.T) {
		// 外部缓存为空，从数据库获取ID批次
		mockExternalCache.EXPECT().GetSingleID(gomock.Any()).Return(uint64(0), errorx.New(errorx.CodeNotFound, "缓存为空"))
		mockDB.EXPECT().GetBatchIDs(gomock.Any(), uint64(10)).Return([]uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, nil)
		mockExternalCache.EXPECT().FillIDs(gomock.Any(), []uint64{2, 3, 4, 5, 6, 7, 8, 9, 10}).Return(nil)

		id, err := seq.NextID(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, uint64(1), id)
	})

	// 测试从已填充的外部缓存获取ID
	t.Run("从已填充的外部缓存获取ID", func(t *testing.T) {
		mockExternalCache.EXPECT().GetSingleID(gomock.Any()).Return(uint64(2), nil)

		id, err := seq.NextID(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, uint64(2), id)
	})
}

// 测试并发获取ID
func TestSequence_ConcurrentGetIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := databaseMock.NewMockSequenceDatabase(ctrl)
	mockExternalCache := cachexMock.NewMockSequenceCache(ctrl)
	localCache := cachex.NewLocalSequenceCache()

	// 添加计数器变量
	var counter uint64

	// 准备批量ID用于测试
	mockIDs := make([]uint64, 1000)
	for i := 0; i < 1000; i++ {
		mockIDs[i] = uint64(i + 1)
	}

	// 设置外部缓存行为
	mockExternalCache.EXPECT().IsOK(gomock.Any()).Return(true).AnyTimes()
	mockExternalCache.EXPECT().GetSingleID(gomock.Any()).DoAndReturn(func(_ context.Context) (uint64, error) {
		return atomic.AddUint64(&counter, 1), nil
	}).AnyTimes()

	seq := NewSequence(mockDB, mockExternalCache, localCache, SequenceOptions{})

	t.Run("并发获取ID", func(t *testing.T) {
		const goroutines = 10
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for i := 0; i < goroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					id, err := seq.NextID(context.Background())
					assert.NoError(t, err)
					assert.NotZero(t, id)
				}
			}()
		}

		wg.Wait()
	})
}

// 测试边界情况：数据库返回空ID列表
func TestSequence_EmptyIDsFromDatabase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := databaseMock.NewMockSequenceDatabase(ctrl)
	mockExternalCache := cachexMock.NewMockSequenceCache(ctrl)
	localCache := cachex.NewLocalSequenceCache()

	seq := &sequence{
		database:       mockDB,
		externalCache:  mockExternalCache,
		localCache:     localCache,
		maxRetries:     3,
		retryBackoff:   50 * time.Millisecond,
		externPatch:    10,
		cacheThreshold: 20,
		localThreshold: 30,
		localPatch:     5,
	}

	seq.externalCacheAvailable.Store(true)

	t.Run("数据库返回空ID列表", func(t *testing.T) {
		mockExternalCache.EXPECT().GetSingleID(gomock.Any()).Return(uint64(0), errorx.New(errorx.CodeNotFound, "缓存为空"))
		mockDB.EXPECT().GetBatchIDs(gomock.Any(), uint64(10)).Return([]uint64{}, nil)

		id, err := seq.NextID(context.Background())

		assert.Error(t, err)
		assert.Zero(t, id)
		assert.True(t, errorx.Is(err, errorx.CodeNotFound))
	})
}
