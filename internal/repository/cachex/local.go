package cachex

//goland:noinspection ALL
import (
	"context"
	"shortener/internal/types/errorx"
	"sync"
)

func NewLocalSequenceCache() *LocalSequenceCache {
	return &LocalSequenceCache{
		ids:   make([]uint64, 0),
		mutex: &sync.Mutex{},
	}
}

type LocalSequenceCache struct {
	ids   []uint64
	mutex *sync.Mutex
}

func (c *LocalSequenceCache) GetSingleID(ctx context.Context) (uint64, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if len(c.ids) == 0 {
		return 0, errorx.New(errorx.CodeNotFound, "no id is available in the local cache")
	}

	// Get the first ID and remove from queue
	id := c.ids[0]
	c.ids = c.ids[1:]

	return id, nil
}

func (c *LocalSequenceCache) FillIDs(ctx context.Context, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Add new IDs to the end of the queue
	c.ids = append(c.ids, ids...)
	return nil
}

func (c *LocalSequenceCache) IsLessThanThreshold(ctx context.Context, threshold int) (bool, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return len(c.ids) < threshold, nil
}

func (c *LocalSequenceCache) IsOK(ctx context.Context) bool {
	return true
}
