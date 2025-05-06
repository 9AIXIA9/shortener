// Package errorx provides enhanced error handling capabilities with error codes,
// stack traces, metadata, proper error chaining and context integration.
// It supports extracting common identifiers from context objects and
// associates them with errors for better debugging and tracing.
package errorx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"
)

// ErrorX implements enhanced error interface with context, code and metadata support.
type ErrorX struct {
	Code  Code         // Application-specific error code
	_     [60]byte     // Cache row padding to prevent pseudo-sharing
	Msg   string       // Human-readable error message
	Cause error        // Original underlying error
	Stack []string     // Call stack trace for debugging
	Meta  *metadataMap // Contextual metadata key-value pairs
	time  time.Time    // Timestamp when the error was created
}

// New creates a new error instance with the specified code and message.
// Stack traces are captured if globally enabled.
//
// Parameters:
//   - code: The error code to assign
//   - msg: The error message describing the problem
//
// Returns:
//   - A new ErrorX instance
//
// Example:
//
//	err := New(CodeNotFound, "User not found")
func New(code Code, msg string) *ErrorX {
	e := errorPool.Get()
	e.Code = code
	e.Msg = msg
	e.time = time.Now()

	if atomic.LoadUint32(&globalStack) == 1 {
		e.Stack = captureStack(1, defaultStackFilters)
	}
	return e
}

// NewWithCause creates a new error with an underlying cause.
// This is a convenience method combining New and WithCause.
//
// Parameters:
//   - code: The error code to assign
//   - msg: The error message describing the problem
//   - cause: The underlying error that caused this error
//
// Returns:
//   - A new ErrorX instance with the specified cause
//
// Example:
//
//	err := NewWithCause(CodeInternal, "Database operation failed", dbErr)
func NewWithCause(code Code, msg string, cause error) *ErrorX {
	return New(code, msg).WithCause(cause)
}

// WithCause sets the underlying cause of this error.
// If err is nil, the cause remains unchanged.
//
// Parameters:
//   - err: The error to set as the underlying cause
//
// Returns:
//   - The same error instance for method chaining
//
// Example:
//
//	err := New(CodeBadRequest, "Parameter validation failed").WithCause(validationErr)
func (e *ErrorX) WithCause(err error) *ErrorX {
	if err != nil {
		e.Cause = err
	}
	return e
}

// WithMeta adds metadata to the error.
// Creates metadata storage if needed.
//
// Parameters:
//   - key: The metadata key
//   - value: The value to associate with the key
//
// Returns:
//   - The same error instance for method chaining
//
// Example:
//
//	err := New(CodeBadRequest, "Invalid parameter").WithMeta("userId", "12345")
func (e *ErrorX) WithMeta(key string, value interface{}) *ErrorX {
	if e.Meta == nil {
		e.Meta = newMetadataMap()
	}
	e.Meta.Set(contextKey(key), value)
	return e
}

// WithContext enriches an error with values extracted from the provided context.
// It looks for common identifiers like requestID, traceID and userID
// in the context and adds them to the error's metadata for better
// observability and debugging.
//
// If ctx is nil, the original error is returned unchanged.
//
// Example:
//
//	err := New(CodeBadRequest, "invalid parameter").WithContext(ctx)
func (e *ErrorX) WithContext(ctx context.Context) *ErrorX {
	if ctx == nil {
		return e
	}

	if e.Meta == nil {
		e.Meta = newMetadataMap()
	}

	// Extract common context values into error metadata
	for _, key := range contextKeys {
		if value, ok := ctx.Value(key).(string); ok {
			e.Meta.Set(key, value)
		}
	}
	return e
}

// Unwrap supports error chain unwrapping for errors.Is/As compatibility.
// This implements the unwrap interface used by errors.Is and errors.As.
//
// Returns:
//   - The underlying cause of this error
func (e *ErrorX) Unwrap() error {
	return e.Cause
}

// Wrap creates an error wrapping an existing error.
// If err is nil, it behaves like New.
// If err is already an ErrorX, it preserves its context while updating code and message.
//
// Parameters:
//   - err: The error to wrap
//   - code: The error code to assign
//   - msg: The error message describing the problem
//
// Returns:
//   - A new ErrorX instance or a clone of the original with updated information
//
// Example:
//
//	err := Wrap(fileErr, CodeInternal, "Failed to read configuration file")
func Wrap(err error, code Code, msg string) *ErrorX {
	if err == nil {
		return New(code, msg)
	}

	// Check if the error is already an ErrorX
	if existing := new(ErrorX); errors.As(err, &existing) {
		return existing.clone(code, msg)
	}

	return New(code, msg).WithCause(err)
}

// Is checks if an error contains the specified code.
// It returns true for nil errors when checking against CodeSuccess.
//
// Parameters:
//   - err: The error to check
//   - code: The code to compare against
//
// Returns:
//   - true if the error contains the specified code, false otherwise
//
// Example:
//
//	if errorx.Is(err, errorx.CodeNotFound) {
//	    // Handle resource not found case
//	}
func Is(err error, code Code) bool {
	if err == nil {
		return code == CodeSuccess
	}
	var ex *ErrorX
	return errors.As(err, &ex) && ex.Code == code
}

// FormatStack returns formatted stack trace information.
//
// Returns:
//   - A formatted string containing the stack trace, or a message indicating stack traces are unavailable
func (e *ErrorX) FormatStack() string {
	if len(e.Stack) == 0 {
		return "Stack trace not enabled or unavailable"
	}

	var result string
	for i, frame := range e.Stack {
		result += fmt.Sprintf("#%02d %s\n", i, frame)
	}
	return result
}

// Detail returns complete error information including stack trace and metadata.
//
// Returns:
//   - A formatted string containing detailed error information for debugging
func (e *ErrorX) Detail() string {
	var result strings.Builder

	// Basic error information
	result.WriteString(fmt.Sprintf("Error: [%d] %s\n", e.Code, e.Msg))
	result.WriteString(fmt.Sprintf("Time: %s\n", e.time.Format(time.RFC3339)))

	// Cause chain
	if e.Cause != nil {
		result.WriteString(fmt.Sprintf("Cause: %v\n", e.Cause))
	}

	// Metadata
	if e.Meta != nil && len(e.Meta.m) > 0 {
		result.WriteString("Metadata:\n")
		e.Meta.RLock()
		for k, v := range e.Meta.m {
			result.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
		}
		e.Meta.RUnlock()
	}

	// Stack trace
	if len(e.Stack) > 0 {
		result.WriteString("Stack trace:\n")
		for i, frame := range e.Stack {
			result.WriteString(fmt.Sprintf("  #%02d %s\n", i, frame))
		}
	}

	return result.String()
}

// PrintErrorTree prints the complete error tree to the provided writer.
//
// Parameters:
//   - err: The error to print
//   - w: The writer to output to
func PrintErrorTree(err error, w io.Writer) error {
	if err == nil {
		return nil
	}

	if w == nil {
		return errors.New("the output writer cannot be nil")
	}

	if _, err := fmt.Fprintln(w, "Error tree:"); err != nil {
		return fmt.Errorf("printing error tree header failed: %w", err)
	}

	// Use access records to prevent circular references
	visited := make(map[uintptr]bool)
	return printErrorNode(err, w, "", true, visited, 0)
}

// Error implements the standard error interface.
// It returns a formatted string containing the error code, message and cause (if available).
//
// Returns:
//   - A formatted error string
func (e *ErrorX) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Msg, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Msg)
}
