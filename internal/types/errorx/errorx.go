// Package errorx provides enhanced error handling capabilities with error codes,
// stack traces, metadata, proper error chaining and context integration.
// It supports extracting common identifiers from context objects and
// associates them with errors for better debugging and tracing.
package errorx

import (
	"crypto/rand"
	"log"
)

var (
	// globalStack controls stack trace generation using atomic operations.
	// Value 1 enables stack tracing, 0 disables it.
	globalStack uint32

	// errorPool manages error object reuse to reduce memory allocations
	errorPool *AdaptivePool

	// defaultStackFilters Default null filtering
	defaultStackFilters []string
)

// init initializes the error package with default configuration
// and cryptographic components.
func init() {
	//use the default configuration
	Initialize()

	// Initialize cryptographic components for potential secure operations
	var aesKey [16]byte
	if _, err := rand.Read(aesKey[:]); err != nil {
		log.Printf("errorx: failed to initialize AES key: %v", err)
	}
}
