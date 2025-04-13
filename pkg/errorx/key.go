// Package errorx provides enhanced error handling capabilities with error codes,
// stack traces, metadata, proper error chaining and context integration.
// It supports extracting common identifiers from context objects and
// associates them with errors for better debugging and tracing.
package errorx

// contextKey is a type alias for string used as keys in context values and metadata.
// Using a custom type instead of bare string prevents key collisions across packages.
type contextKey string

// Standard context keys for common identifiers used in request processing.
const (
	// RequestIDKey is the key for storing and retrieving request identifiers
	RequestIDKey contextKey = "requestID"

	// TraceIDKey is the key for storing and retrieving distributed tracing identifiers
	TraceIDKey contextKey = "traceID"

	// UserIDKey is the key for storing and retrieving user identifiers
	UserIDKey contextKey = "userID"
)

// contextKeys the key of information recorded by Meta
var contextKeys = []contextKey{
	RequestIDKey,
	TraceIDKey,
	UserIDKey,
}
