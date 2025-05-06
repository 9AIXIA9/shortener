// Package errorx provides enhanced error handling capabilities with error codes,
// stack traces, metadata, proper error chaining and context integration.
// It supports extracting common identifiers from context objects and
// associates them with errors for better debugging and tracing.
package errorx

import "net/http"

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
	CodeTimeout
	CodeServiceUnavailable
	CodeTooFrequent
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
		return http.StatusOK // OK
	case CodeParamError:
		return http.StatusBadRequest // Bad Request
	case CodeNotFound:
		return http.StatusNotFound // Not Found
	case CodeDatabaseError, CodeCacheError, CodeSystemError:
		return http.StatusInternalServerError // Internal Server Error
	case CodeServiceUnavailable:
		return http.StatusServiceUnavailable // Service Unavailable
	case CodeTimeout:
		return http.StatusRequestTimeout
	case CodeTooFrequent:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError // Default to Internal Server Error
	}
}
