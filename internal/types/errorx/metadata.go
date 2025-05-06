// Package errorx provides enhanced error handling capabilities with error codes,
// stack traces, metadata, proper error chaining and context integration.
// It supports extracting common identifiers from context objects and
// associates them with errors for better debugging and tracing.
package errorx

import "sync"

// metadataMap provides a concurrent-safe storage for error metadata.
// It wraps a standard map with mutex protection for thread safety.
type metadataMap struct {
	sync.RWMutex
	m map[contextKey]interface{}
}

// newMetadataMap creates and initializes a new metadata storage container.
// The initial capacity is set to 4 to optimize for common use cases.
//
// Returns:
//   - A new initialized metadataMap instance
func newMetadataMap() *metadataMap {
	return &metadataMap{m: make(map[contextKey]interface{}, 4)}
}

// Set stores a key-value pair in the metadata map.
// This operation is thread-safe.
//
// Parameters:
//   - key: The identifier for the metadata
//   - value: The data to associate with the key
func (m *metadataMap) Set(key contextKey, value interface{}) {
	m.Lock()
	defer m.Unlock()
	m.m[key] = value
}

// Get retrieves a value by its key from the metadata map.
// This operation is thread-safe.
//
// Parameters:
//   - key: The identifier to look up
//
// Returns:
//   - The associated value and true if found
//   - nil and false if not found
func (m *metadataMap) Get(key contextKey) (interface{}, bool) {
	m.RLock()
	defer m.RUnlock()
	v, ok := m.m[key]
	return v, ok
}

// Clear removes all entries from the metadata map and resets it
// to an empty map with initial capacity of 4.
// This operation is thread-safe.
func (m *metadataMap) Clear() {
	m.Lock()
	defer m.Unlock()
	m.m = make(map[contextKey]interface{}, 4)
}

// Copy creates a deep copy of the metadata map.
// The returned map is a new instance with copied values.
// This operation is thread-safe.
//
// Returns:
//   - A new metadataMap instance with the same content
func (m *metadataMap) Copy() *metadataMap {
	m.RLock()
	defer m.RUnlock()
	cpy := newMetadataMap()
	for k, v := range m.m {
		cpy.m[k] = v
	}
	return cpy
}
