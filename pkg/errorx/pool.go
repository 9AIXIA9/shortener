// Package errorx provides enhanced error handling capabilities with error codes,
// stack traces, metadata, proper error chaining and context integration.
// It supports extracting common identifiers from context objects and
// associates them with errors for better debugging and tracing.
package errorx

import (
	"sync"
	"time"
)

// AdaptivePool manages error object recycling with dynamic capacity adjustment.
// It combines a channel-based buffer with a fallback sync.Pool for optimal performance.
type AdaptivePool struct {
	mu           sync.Mutex   // Mutex that protects concurrent access
	recycleChan  chan *ErrorX //  Main channel for recycling error objects
	fallbackPool *sync.Pool   //  Fallback pool used when the main channel is full
	config       PoolConfig   // Pool configuration parameters
	closed       bool         // Indicates whether the pool is closed
}

// PoolConfig defines configuration parameters for the error object pool.
type PoolConfig struct {
	// MaxSize specifies the maximum number of objects that can be stored in the pool.
	MaxSize int

	// BufferSize defines the initial buffer capacity for the recycling channel.
	BufferSize int

	// MonitorCycle determines how often the pool capacity is evaluated for adjustment.
	MonitorCycle time.Duration
}

// NewAdaptivePool creates and initializes a managed error object pool.
// It starts a background goroutine to monitor and adjust pool capacity.
//
// Parameters:
//   - cfg: Configuration parameters for the pool
//
// Returns:
//   - A new initialized AdaptivePool instance
func NewAdaptivePool(cfg PoolConfig) *AdaptivePool {
	ap := &AdaptivePool{
		recycleChan: make(chan *ErrorX, cfg.BufferSize),
		fallbackPool: &sync.Pool{
			New: func() interface{} {
				return &ErrorX{Meta: newMetadataMap()}
			},
		},
		config: cfg,
	}
	go ap.monitor()
	return ap
}

// Get retrieves an error object from the pool or creates a new one if the pool is empty.
// This method is thread-safe.
//
// Returns:
//   - An initialized ErrorX object ready for use
func (ap *AdaptivePool) Get() *ErrorX {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if ap.closed {
		return ap.fallbackPool.Get().(*ErrorX)
	}

	select {
	case e := <-ap.recycleChan:
		return e
	default:
		return ap.fallbackPool.Get().(*ErrorX)
	}
}

// Put recycles an error object back to the pool after resetting its state.
// If the primary channel is full, it falls back to the sync.Pool.
// This method is thread-safe.
//
// Parameters:
//   - e: The error object to recycle
func (ap *AdaptivePool) Put(e *ErrorX) {
	if e == nil {
		return
	}

	ap.mu.Lock()
	defer ap.mu.Unlock()

	if ap.closed {
		return
	}

	e.reset()

	select {
	case ap.recycleChan <- e:
		// successfully placed in the recycleChan
	default:
		ap.fallbackPool.Put(e)
	}
}

// monitor runs in a separate goroutine to periodically adjust the pool capacity
// based on usage patterns.
func (ap *AdaptivePool) monitor() {
	ticker := time.NewTicker(ap.config.MonitorCycle)
	defer ticker.Stop()

	for range ticker.C {
		if ap.isClosed() {
			return
		}
		ap.adjustCapacity()
	}
}

// isClosed checks if the pool has been closed.
// This method is thread-safe.
func (ap *AdaptivePool) isClosed() bool {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	return ap.closed
}

// Close shuts down the pool and prevents further recycling of objects.
// This method is thread-safe and idempotent.
func (ap *AdaptivePool) Close() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if ap.closed {
		return
	}

	ap.closed = true
	close(ap.recycleChan)
}

// adjustCapacity implements dynamic pool size adjustments based on usage.
// It expands capacity when utilization exceeds 80% up to the configured maximum size.
func (ap *AdaptivePool) adjustCapacity() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if ap.closed {
		return
	}

	current := len(ap.recycleChan)
	capacity := cap(ap.recycleChan)

	// Expand capacity when utilization exceeds 80%
	if float64(current)/float64(capacity) > 0.8 && capacity < ap.config.MaxSize {
		newCap := minInt(capacity*2, ap.config.MaxSize)
		newChan := make(chan *ErrorX, newCap)

		// securely transfer objects to a new channel
		oldChan := ap.recycleChan
		ap.recycleChan = newChan

		// transfer an existing object after converting a channel
		go func(old chan *ErrorX) {
			for len(old) > 0 {
				select {
				case e := <-old:
					ap.Put(e)
				default:
					return
				}
			}
		}(oldChan)
	}

	// update the pool usage metric
	//metrics.poolUtilization.Set(float64(current) / float64(capacity))
}
