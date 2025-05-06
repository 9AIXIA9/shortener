// Package errorx provides enhanced error handling capabilities with error codes,
// stack traces, metadata, proper error chaining and context integration.
// It supports extracting common identifiers from context objects and
// associates them with errors for better debugging and tracing.
package errorx

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

//todo: read

// captureStack captures call stack traces up to a maximum depth.
// It returns a slice of strings representing each stack frame.
//
// Parameters:
//   - skip: The number of stack frames to skip from the top of the call stack
//   - filters: List of package prefixes to exclude from the stack trace
//
// Returns:
//   - A slice of formatted stack frame strings, or nil if no frames were captured
func captureStack(skip int, filters []string) []string {
	const maxDepth = 32
	pc := make([]uintptr, maxDepth)
	n := runtime.Callers(skip+2, pc)
	if n == 0 {
		return nil
	}

	frames := runtime.CallersFrames(pc[:n])
	stack := make([]string, 0, n)

	for {
		frame, more := frames.Next()

		// Check if this frame should be filtered
		skipFrame := false
		for _, pkg := range filters {
			if strings.HasPrefix(frame.Function, pkg) {
				skipFrame = true
				break
			}
		}

		if !skipFrame {
			stack = append(stack, fmt.Sprintf("%s:%d %s",
				frame.File, frame.Line, frame.Function))
		}

		if !more {
			break
		}
	}
	return stack
}

// printErrorNode writes a formatted representation of an error to the provided writer.
// It handles ErrorX objects specially, printing their code, message, metadata, and cause.
// The function builds a tree-like structure to represent nested errors.
//
// Parameters:
//   - err: The error to print
//   - w: Writer to output the formatted error
//   - indent: Current indentation string for formatting
//   - isLast: Whether this is the last error in the current branch
func printErrorNode(err error, w io.Writer, indent string, isLast bool,
	visited map[uintptr]bool, depth int) error {
	// Prevent stack overflow
	const maxDepth = 100
	if depth > maxDepth {
		_, err := fmt.Fprintf(w, "%s... (Exceeding the maximum depth)\n", indent)
		return err
	}

	// Detect circular references
	var ex2 *ErrorX
	if errors.As(err, &ex2) {
		ptr := reflect.ValueOf(ex2).Pointer()
		if visited[ptr] {
			_, err := fmt.Fprintf(w, "%s└── (A circular reference was detected)\n", indent)
			return err
		}
		visited[ptr] = true
	}

	marker := "└── "
	if !isLast {
		marker = "├── "
	}

	nextIndent := indent + "    "
	if !isLast {
		nextIndent = indent + "│   "
	}

	// Print the current error
	var ex *ErrorX
	if errors.As(err, &ex) {
		if _, err := fmt.Fprintf(w, "%s%s[%d] %s\n", indent, marker, ex.Code, ex.Msg); err != nil {
			return err
		}

		// Print metadata
		if ex.Meta != nil && len(ex.Meta.m) > 0 {
			if _, err := fmt.Fprintf(w, "%s└── Metadata:\n", nextIndent); err != nil {
				return err
			}

			metaIndent := nextIndent + "    "

			// Handle metadata securely
			ex.Meta.RLock()
			keys := make([]contextKey, 0, len(ex.Meta.m))
			values := make([]interface{}, 0, len(ex.Meta.m))

			for k, v := range ex.Meta.m {
				keys = append(keys, k)
				values = append(values, v)
			}
			ex.Meta.RUnlock()

			// Print metadata (already out of lock)
			for i, k := range keys {
				isLastMeta := i == len(keys)-1
				metaMarker := "└── "
				if !isLastMeta {
					metaMarker = "├── "
				}

				// Safe handling values
				vStr := fmt.Sprintf("%v", values[i])
				if _, err := fmt.Fprintf(w, "%s%s%s: %s\n", metaIndent, metaMarker, k, vStr); err != nil {
					return err
				}
			}
		}

		// Handle the chain of causes
		if ex.Cause != nil {
			return printErrorNode(ex.Cause, w, nextIndent, true, visited, depth+1)
		}
	} else {
		// Handle non-ErrorX type errors
		if _, err := fmt.Fprintf(w, "%s%s%v\n", indent, marker, err); err != nil {
			return err
		}
	}

	return nil
}

// minInt returns the smaller of two integers.
// This is a helper function since min() is only available in Go 1.21+.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// reset prepares the error instance for reuse by clearing all fields.
// This is an internal method used by the error pool.
func (e *ErrorX) reset() {
	e.Code = 0
	e.Msg = ""
	e.Cause = nil
	e.Stack = nil
	e.time = time.Time{}
	if e.Meta != nil {
		e.Meta.Clear()
	}
}

// clone creates a copy of this error with updated code and message.
// It preserves the original error as the cause.
//
// Parameters:
//   - code: The new error code
//   - msg: The new error message
//
// Returns:
//   - A new ErrorX instance with the original error as its cause
func (e *ErrorX) clone(code Code, msg string) *ErrorX {
	newErr := errorPool.Get()
	newErr.Code = code
	newErr.Msg = msg
	newErr.Cause = e
	newErr.time = time.Now()

	if e.Meta != nil {
		newErr.Meta = e.Meta.Copy()
	}

	if atomic.LoadUint32(&globalStack) == 1 {
		newErr.Stack = captureStack(1, defaultStackFilters)
	}

	return newErr
}
