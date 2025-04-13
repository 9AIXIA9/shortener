// Package errorx provides enhanced error handling capabilities with error codes,
// stack traces, metadata, proper error chaining and context integration.
// It supports extracting common identifiers from context objects and
// associates them with errors for better debugging and tracing.
package errorx

import (
	"github.com/prometheus/client_golang/prometheus"
)

// metrics contains all prometheus metrics for the errorx package.
// These metrics help monitor the performance and behavior of the error pool.
var metrics = struct {
	// poolOperations tracks the number of Get/Put operations performed on the error pool,
	// labeled by operation type ("get" or "put").
	poolOperations *prometheus.CounterVec

	// poolUtilization tracks the current utilization ratio of the error object pool
	// as a value between 0.0 and 1.0.
	poolUtilization prometheus.Gauge
}{
	poolOperations: prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "errorx",
			Subsystem: "pool",
			Name:      "operations_total",
			Help:      "Number of memory pool operations by type (get/put)",
		},
		[]string{"operation"},
	),
	poolUtilization: prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "errorx",
			Subsystem: "pool",
			Name:      "utilization_ratio",
			Help:      "Current pool utilization ratio (0.0-1.0)",
		},
	),
}

// RegisterMetrics registers all errorx metrics with the provided prometheus registerer.
// This should be called during application initialization to enable metric collection.
//
// Parameters:
//   - reg: The prometheus.Registerer where metrics should be registered
//
// Example:
//
//	func main() {
//		errorx.RegisterMetrics(prometheus.DefaultRegisterer)
//	}
func RegisterMetrics(reg prometheus.Registerer) {
	reg.MustRegister(
		metrics.poolOperations,
		metrics.poolUtilization,
	)
}
