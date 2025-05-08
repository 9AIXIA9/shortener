package cachex

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"shortener/internal/types/errorx"
	"sync"
)

const (
	defaultCap = 1000
)

// NewLocalSequenceCache 创建一个新的本地序列缓存
func NewLocalSequenceCache(capacity int) *LocalSequenceCache {
	if capacity <= 0 {
		capacity = defaultCap
	}
	return &LocalSequenceCache{
		head:  0,
		tail:  0,
		ids:   make([]uint64, capacity+1), // 环形缓冲区需要额外空间
		cap:   capacity,
		mutex: &sync.RWMutex{},
	}
}

type LocalSequenceCache struct {
	head  int           // 队列头指针
	tail  int           // 队列尾指针
	ids   []uint64      // 环形缓冲区
	cap   int           // 容量
	mutex *sync.RWMutex // 读写锁
}

// GetSingleID 获取单个ID
func (c *LocalSequenceCache) GetSingleID(ctx context.Context) (id uint64, err error) {
	return id, ProcessTimeout(ctx, func() error {
		c.mutex.Lock()
		defer c.mutex.Unlock()

		if c.head == c.tail {
			return errorx.New(errorx.CodeNotFound, "no id is available in the local cache")
		}

		// 获取ID并移动头指针
		id = c.ids[c.head]
		c.head = (c.head + 1) % (c.cap + 1)

		return nil
	})
}

// FillIDs 填充ID到缓存
func (c *LocalSequenceCache) FillIDs(ctx context.Context, ids []uint64) error {
	if len(ids) == 0 {
		return nil // 空列表直接返回成功
	}

	return ProcessTimeout(ctx, func() error {
		c.mutex.Lock()
		defer c.mutex.Unlock()

		// 计算可用空间
		available := 0
		if c.head <= c.tail {
			available = c.cap - (c.tail - c.head)
		} else {
			available = c.head - c.tail - 1
		}

		// 确定能填充的ID数量
		toLoad := len(ids)
		if toLoad > available {
			toLoad = available
			logx.Infof("local cache capacity insufficient, only filling %v ids", toLoad)
		}

		// 填充ID
		for i := 0; i < toLoad; i++ {
			c.ids[c.tail] = ids[i]
			c.tail = (c.tail + 1) % (c.cap + 1)
		}

		logx.Infof("%v ids successfully filled, %v ids not filled", toLoad, len(ids)-toLoad)
		return nil
	})
}

// IsLessThanThreshold 检查当前缓存中的ID数量是否小于阈值
func (c *LocalSequenceCache) IsLessThanThreshold(ctx context.Context, threshold int) (bool, error) {
	var result bool
	err := ProcessTimeout(ctx, func() error {
		c.mutex.RLock() // 只读操作使用读锁
		defer c.mutex.RUnlock()

		result = c.length() < threshold
		return nil
	})

	return result, err
}

// IsOK 检查缓存是否正常工作
func (c *LocalSequenceCache) IsOK(ctx context.Context) bool {
	return ProcessTimeout(ctx, func() error {
		return nil
	}) == nil
}

// 计算缓存中的元素数量
func (c *LocalSequenceCache) length() int {
	if c.tail >= c.head {
		return c.tail - c.head
	}
	return (c.cap + 1) - (c.head - c.tail)
}

// ProcessTimeout 处理带超时的操作
func ProcessTimeout(ctx context.Context, f func() error) error {
	ch := make(chan error, 1)

	go func() {
		ch <- f()
		close(ch)
	}()

	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		return errorx.New(errorx.CodeTimeout, "the operation timed out")
	}
}
