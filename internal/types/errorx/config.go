// Package errorx provides enhanced error handling capabilities with error codes,
// stack traces, metadata, proper error chaining and context integration.
// It supports extracting common identifiers from context objects and
// associates them with errors for better debugging and tracing.
package errorx

import (
	"sync/atomic"
	"time"
)

// config defines global error handling configuration parameters.
type config struct {
	// poolSize specifies initial error object pool capacity
	poolSize int
	// maxPoolSize defines the maximum growth limit for the pool
	maxPoolSize int
	// enableStack controls whether stack traces are captured
	enableStack bool
	// monitorInterval sets frequency of pool capacity adjustments
	monitorInterval time.Duration
	// stackFilters defines package prefixes to be filtered out from stack traces
	stackFilters []string
}

// Initialize configures global error handling settings.
// It sets up the error object pool with the specified options.
// If no options are provided, default values are used.
//
// Parameters:
//   - opts: A variadic list of Option functions to configure the package behavior
func Initialize(opts ...Option) {
	cfg := &config{
		poolSize:        512,
		maxPoolSize:     1024,
		enableStack:     true,
		monitorInterval: 30 * time.Second,
		stackFilters:    []string{}, // default empty filter list
	}

	for _, opt := range opts {
		opt(cfg)
	}

	EnableStackTracing(cfg.enableStack)

	defaultStackFilters = cfg.stackFilters

	errorPool = NewAdaptivePool(PoolConfig{
		MaxSize:      cfg.maxPoolSize,
		BufferSize:   cfg.poolSize,
		MonitorCycle: cfg.monitorInterval,
	})
}

// Option configures library behavior through functional options pattern.
type Option func(*config)

// WithPoolSize sets initial pool capacity.
//
// Parameters:
//   - size: The initial capacity of the error object pool
//
// Returns:
//   - An Option function that modifies the configuration
func WithPoolSize(size int) Option {
	return func(c *config) {
		if size > 0 {
			c.poolSize = size
		}
	}
}

// WithMaxPoolSize sets maximum pool growth limit.
//
// Parameters:
//   - size: The maximum number of error objects that can be stored in the pool
//
// Returns:
//   - An Option function that modifies the configuration
func WithMaxPoolSize(size int) Option {
	return func(c *config) {
		if size > 0 {
			c.maxPoolSize = size
		}
	}
}

// WithStackTracing enables or disables stack trace collection.
//
// Parameters:
//   - enabled: Whether to capture stack traces with errors
//
// Returns:
//   - An Option function that modifies the configuration
func WithStackTracing(enabled bool) Option {
	return func(c *config) {
		c.enableStack = enabled
	}
}

// WithMonitorInterval sets the frequency for pool capacity adjustments.
//
// Parameters:
//   - interval: The time duration between pool monitoring checks
//
// Returns:
//   - An Option function that modifies the configuration
func WithMonitorInterval(interval time.Duration) Option {
	return func(c *config) {
		if interval > 0 {
			c.monitorInterval = interval
		}
	}
}

// WithStackFilters specifies package prefixes to be excluded from stack traces.
// This helps reduce noise in stack traces by removing frames from specified packages.
//
// Parameters:
//   - packages: A variadic list of package name prefixes to filter out
//
// Returns:
//   - An Option function that modifies the configuration
func WithStackFilters(packages ...string) Option {
	return func(c *config) {
		c.stackFilters = packages
	}
}

// EnableStackTracing controls whether error objects capture stack traces.
// When enabled, error objects include stack information to aid debugging.
// When disabled, errors use less memory and perform better.
func EnableStackTracing(enabled bool) {
	var val uint32
	if enabled {
		val = 1
	}
	atomic.StoreUint32(&globalStack, val)
}
