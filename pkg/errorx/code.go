// Package errorx provides enhanced error handling capabilities with error codes,
// stack traces, metadata, proper error chaining and context integration.
// It supports extracting common identifiers from context objects and
// associates them with errors for better debugging and tracing.
package errorx

// Code represents application-specific error codes.
// These codes help categorize errors and provide consistent error responses.
type Code int32

// Standard error codes for common scenarios.
const (
	CodeSuccess Code = 1000 + iota
	CodeSystemError
	CodeDatabaseError
	CodeCacheError

	CodeParamError
	CodeNotFound
	CodeServiceUnavailable
)

// ToHTTPStatus maps an application error code to the appropriate HTTP status code.
// This allows consistent HTTP responses based on internal error classifications.
//
// Parameters:
//   - code: The application-specific error code
//
// Returns:
//   - The corresponding HTTP status code
func ToHTTPStatus(code Code) int {
	switch code {
	case CodeSuccess:
		return 200 // OK
	case CodeParamError:
		return 400 // Bad Request
	case CodeNotFound:
		return 404 // Not Found
	case CodeDatabaseError, CodeCacheError, CodeSystemError:
		return 500 // Internal Server Error
	case CodeServiceUnavailable:
		return 503 // Service Unavailable
	default:
		return 500 // Default to Internal Server Error
	}
}
