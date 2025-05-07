package cachex

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

func (c *LocalSequenceCache) GetSingleID(ctx context.Context) (id uint64, err error) {
	if err = processTimeout(ctx, func() error {
		c.mutex.Lock()
		defer c.mutex.Unlock()

		if len(c.ids) == 0 {
			return errorx.New(errorx.CodeNotFound, "no id is available in the local cache")
		}

		// Get the first ID and remove from queue
		id = c.ids[0]
		c.ids = c.ids[1:]

		return nil
	}); err != nil {
		return 0, errorx.Wrap(err, errorx.CodeTimeout, "get single id failed")
	}

	return id, nil
}

func (c *LocalSequenceCache) FillIDs(ctx context.Context, ids []uint64) error {
	if err := processTimeout(ctx, func() error {
		if len(ids) == 0 {
			return nil
		}

		c.mutex.Lock()
		defer c.mutex.Unlock()

		// Add new IDs to the end of the queue
		c.ids = append(c.ids, ids...)
		return nil
	}); err != nil {
		return errorx.Wrap(err, errorx.CodeTimeout, "fill ids failed")
	}

	return nil
}

func (c *LocalSequenceCache) IsLessThanThreshold(ctx context.Context, threshold int) (ok bool, err error) {
	if err = processTimeout(ctx, func() error {
		c.mutex.Lock()
		defer c.mutex.Unlock()

		ok = len(c.ids) < threshold

		return nil
	}); err != nil {
		return false, errorx.Wrap(err, errorx.CodeTimeout, "check if is less than threshold failed")
	}

	return ok, nil
}

func (c *LocalSequenceCache) IsOK(ctx context.Context) bool {
	if err := processTimeout(ctx, func() error {
		return nil
	}); err != nil {
		return false
	}

	return true
}

func processTimeout(ctx context.Context, f func() error) error {
	done := make(chan error, 1)

	go func() {
		done <- f()
	}()

	select {
	case <-ctx.Done():
		return errorx.New(errorx.CodeTimeout, "操作超时")
	case err := <-done:
		return err
	}
}
