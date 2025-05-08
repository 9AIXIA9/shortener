package cachex

import (
	"context"
	"shortener/internal/types/errorx"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 测试创建本地缓存
func TestNewLocalSequenceCache(t *testing.T) {
	t.Run("默认容量", func(t *testing.T) {
		cache := NewLocalSequenceCache(0)
		assert.Equal(t, defaultCap, cache.cap)
		assert.Equal(t, defaultCap+1, len(cache.ids))
	})

	t.Run("自定义容量", func(t *testing.T) {
		capacity := 100
		cache := NewLocalSequenceCache(capacity)
		assert.Equal(t, capacity, cache.cap)
		assert.Equal(t, capacity+1, len(cache.ids))
	})
}

// 测试从空缓存获取ID
func TestLocalSequenceCache_GetSingleID_Empty(t *testing.T) {
	cache := NewLocalSequenceCache(10)
	ctx := context.Background()

	_, err := cache.GetSingleID(ctx)
	assert.Error(t, err)
	var targetErr *errorx.ErrorX
	assert.ErrorAs(t, err, &targetErr)
	assert.Equal(t, errorx.CodeNotFound, targetErr.Code)
}

// 测试填充ID后获取ID
func TestLocalSequenceCache_FillAndGetID(t *testing.T) {
	cache := NewLocalSequenceCache(10)
	ctx := context.Background()

	// 填充一些ID
	ids := []uint64{1, 2, 3, 4, 5}
	err := cache.FillIDs(ctx, ids)
	assert.NoError(t, err)

	// 检查缓存长度
	lessThanSix, err := cache.IsLessThanThreshold(ctx, 6)
	assert.NoError(t, err)
	assert.True(t, lessThanSix)

	// 逐个获取ID
	for _, expected := range ids {
		id, err := cache.GetSingleID(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expected, id)
	}

	// 再次从空缓存获取
	_, err = cache.GetSingleID(ctx)
	assert.Error(t, err)
	var targetErr *errorx.ErrorX
	assert.ErrorAs(t, err, &targetErr)
	assert.Equal(t, errorx.CodeNotFound, targetErr.Code)

	ids = append(ids, 6, 7, 8, 9)
	err = cache.FillIDs(ctx, ids)
	assert.NoError(t, err)

	// 检查缓存长度
	lessThanSix, err = cache.IsLessThanThreshold(ctx, 6)
	assert.NoError(t, err)
	assert.False(t, lessThanSix)
}

// 测试填充超过容量的ID
func TestLocalSequenceCache_FillIDs_ExceedCapacity(t *testing.T) {
	capacity := 5
	cache := NewLocalSequenceCache(capacity)
	ctx := context.Background()

	// 填充超过容量的ID
	ids := []uint64{1, 2, 3, 4, 5, 6, 7, 8}
	err := cache.FillIDs(ctx, ids)
	assert.NoError(t, err)

	// 验证只能获取容量个ID
	for i := 0; i < capacity; i++ {
		_, err := cache.GetSingleID(ctx)
		assert.NoError(t, err)
	}

	// 再次获取应返回错误
	_, err = cache.GetSingleID(ctx)
	assert.Error(t, err)
}

// 测试环形缓冲区的环绕
func TestLocalSequenceCache_CircularBuffer(t *testing.T) {
	capacity := 3
	cache := NewLocalSequenceCache(capacity)
	ctx := context.Background()

	// 第一轮填充
	err := cache.FillIDs(ctx, []uint64{1, 2, 3})
	assert.NoError(t, err)

	// 获取两个ID
	id1, err := cache.GetSingleID(ctx)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), id1)

	id2, err := cache.GetSingleID(ctx)
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), id2)

	// 再填充两个ID
	err = cache.FillIDs(ctx, []uint64{4, 5})
	assert.NoError(t, err)

	// 获取剩下的ID
	id3, err := cache.GetSingleID(ctx)
	assert.NoError(t, err)
	assert.Equal(t, uint64(3), id3)

	id4, err := cache.GetSingleID(ctx)
	assert.NoError(t, err)
	assert.Equal(t, uint64(4), id4)

	id5, err := cache.GetSingleID(ctx)
	assert.NoError(t, err)
	assert.Equal(t, uint64(5), id5)

	// 缓存为空
	_, err = cache.GetSingleID(ctx)
	assert.Error(t, err)
}

// 测试IsLessThanThreshold方法
func TestLocalSequenceCache_IsLessThanThreshold(t *testing.T) {
	cache := NewLocalSequenceCache(10)
	ctx := context.Background()

	// 空缓存
	result, err := cache.IsLessThanThreshold(ctx, 1)
	assert.NoError(t, err)
	assert.True(t, result)

	// 填充ID
	err = cache.FillIDs(ctx, []uint64{1, 2, 3, 4, 5})
	assert.NoError(t, err)

	// 测试阈值小于当前数量
	result, err = cache.IsLessThanThreshold(ctx, 3)
	assert.NoError(t, err)
	assert.False(t, result)

	// 测试阈值等于当前数量
	result, err = cache.IsLessThanThreshold(ctx, 5)
	assert.NoError(t, err)
	assert.False(t, result)

	// 测试阈值大于当前数量
	result, err = cache.IsLessThanThreshold(ctx, 10)
	assert.NoError(t, err)
	assert.True(t, result)
}

// 测试IsOK方法
func TestLocalSequenceCache_IsOK(t *testing.T) {
	cache := NewLocalSequenceCache(10)
	ctx := context.Background()

	assert.True(t, cache.IsOK(ctx))

	// 测试超时上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
	defer cancel()
	time.Sleep(5 * time.Millisecond)

	assert.False(t, cache.IsOK(timeoutCtx))
}

// 测试超时处理
func TestLocalSequenceCache_Timeout(t *testing.T) {
	cache := NewLocalSequenceCache(10)

	// 创建一个已超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(5 * time.Millisecond)

	_, err := cache.GetSingleID(ctx)
	assert.Error(t, err)
	var targetErr *errorx.ErrorX
	assert.ErrorAs(t, err, &targetErr)
	assert.Equal(t, errorx.CodeTimeout, targetErr.Code)

	err = cache.FillIDs(ctx, []uint64{1, 2, 3})
	assert.Error(t, err)
	assert.ErrorAs(t, err, &targetErr)
	assert.Equal(t, errorx.CodeTimeout, targetErr.Code)

	_, err = cache.IsLessThanThreshold(ctx, 5)
	assert.Error(t, err)
	assert.ErrorAs(t, err, &targetErr)
	assert.Equal(t, errorx.CodeTimeout, targetErr.Code)
}

// 测试并发访问
func TestLocalSequenceCache_Concurrent(t *testing.T) {
	cache := NewLocalSequenceCache(500)
	ctx := context.Background()

	// 填充ID
	ids := make([]uint64, 500)
	for i := range ids {
		ids[i] = uint64(i + 1)
	}
	err := cache.FillIDs(ctx, ids)
	assert.NoError(t, err)

	// 并发获取ID
	var wg sync.WaitGroup
	results := sync.Map{}
	goroutines := 10
	idsPerGoroutine := 50

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id, err := cache.GetSingleID(ctx)
				if err != nil {
					t.Errorf("获取ID错误: %v", err)
					return
				}

				if _, loaded := results.LoadOrStore(id, true); loaded {
					t.Errorf("重复ID: %v", id)
				}
			}
		}()
	}

	wg.Wait()

	// 验证获取的ID数量
	count := 0
	results.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, goroutines*idsPerGoroutine, count)
}

// 测试填充空ID列表
func TestLocalSequenceCache_FillIDs_Empty(t *testing.T) {
	cache := NewLocalSequenceCache(10)
	ctx := context.Background()

	err := cache.FillIDs(ctx, []uint64{})
	assert.NoError(t, err)

	// 验证缓存仍为空
	_, err = cache.GetSingleID(ctx)
	assert.Error(t, err)
}
