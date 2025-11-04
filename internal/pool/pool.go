// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package pool

import (
	"bytes"
	"sync"
)

const (
	// Buffer size thresholds for pooling
	maxBufferPoolSize        = 64 * 1024 // 64KB - max buffer size to pool
	maxStringBuilderPoolSize = 32 * 1024 // 32KB - max string builder size to pool

	// JSON buffer size thresholds
	smallJSONThreshold  = 1024      // 1KB - threshold for small JSON
	mediumJSONThreshold = 64 * 1024 // 64KB - threshold for medium JSON

	// Map pool settings
	maxMapPoolSize = 100 // Max map entries to pool
)

// BufferPool is a pool of reusable byte buffers
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool creates a new buffer pool
func NewBufferPool() *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

// Get retrieves a buffer from the pool
func (p *BufferPool) Get() *bytes.Buffer {
	buf := p.pool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// Put returns a buffer to the pool
func (p *BufferPool) Put(buf *bytes.Buffer) {
	// Only pool buffers under max size to avoid holding large buffers
	if buf.Cap() < maxBufferPoolSize {
		p.pool.Put(buf)
	}
}

// ByteSlicePool is a pool of reusable byte slices
type ByteSlicePool struct {
	pool sync.Pool
	size int
}

// NewByteSlicePool creates a new byte slice pool with a specific size
func NewByteSlicePool(size int) *ByteSlicePool {
	return &ByteSlicePool{
		size: size,
		pool: sync.Pool{
			New: func() interface{} {
				slice := make([]byte, size)
				return &slice
			},
		},
	}
}

// Get retrieves a byte slice from the pool
func (p *ByteSlicePool) Get() []byte {
	slicePtr := p.pool.Get().(*[]byte)
	return (*slicePtr)[:p.size]
}

// Put returns a byte slice to the pool
func (p *ByteSlicePool) Put(slice []byte) {
	if cap(slice) >= p.size {
		slice = slice[:p.size]
		p.pool.Put(&slice)
	}
}

// JSONBufferPool is specialized for JSON encoding/decoding
type JSONBufferPool struct {
	small  *BufferPool // For small JSON (< 1KB)
	medium *BufferPool // For medium JSON (1KB - 64KB)
	large  *BufferPool // For large JSON (> 64KB)
}

// NewJSONBufferPool creates a new JSON buffer pool
func NewJSONBufferPool() *JSONBufferPool {
	return &JSONBufferPool{
		small:  NewBufferPool(),
		medium: NewBufferPool(),
		large:  NewBufferPool(),
	}
}

// Get retrieves an appropriately sized buffer
func (p *JSONBufferPool) Get(estimatedSize int) *bytes.Buffer {
	switch {
	case estimatedSize < smallJSONThreshold:
		return p.small.Get()
	case estimatedSize < mediumJSONThreshold:
		return p.medium.Get()
	default:
		return p.large.Get()
	}
}

// Put returns a buffer to the appropriate pool
func (p *JSONBufferPool) Put(buf *bytes.Buffer) {
	size := buf.Len()
	switch {
	case size < smallJSONThreshold:
		p.small.Put(buf)
	case size < mediumJSONThreshold:
		p.medium.Put(buf)
	default:
		p.large.Put(buf)
	}
}

// MapPool is a pool for map[string]interface{}
type MapPool struct {
	pool sync.Pool
}

// NewMapPool creates a new map pool
func NewMapPool() *MapPool {
	return &MapPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make(map[string]interface{}, 10)
			},
		},
	}
}

// Get retrieves a map from the pool
func (p *MapPool) Get() map[string]interface{} {
	m := p.pool.Get().(map[string]interface{})
	// Clear the map
	for k := range m {
		delete(m, k)
	}
	return m
}

// Put returns a map to the pool
func (p *MapPool) Put(m map[string]interface{}) {
	// Only pool small maps
	if len(m) < maxMapPoolSize {
		p.pool.Put(m)
	}
}

// StringBuilderPool is a pool for strings.Builder
type StringBuilderPool struct {
	pool sync.Pool
}

// NewStringBuilderPool creates a new string builder pool
func NewStringBuilderPool() *StringBuilderPool {
	return &StringBuilderPool{
		pool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

// Get retrieves a buffer from the pool (used as StringBuilder)
func (p *StringBuilderPool) Get() *bytes.Buffer {
	buf := p.pool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// Put returns a buffer to the pool
func (p *StringBuilderPool) Put(buf *bytes.Buffer) {
	if buf.Cap() < maxStringBuilderPoolSize {
		p.pool.Put(buf)
	}
}

// GlobalPools provides access to commonly used pools
var GlobalPools = struct {
	Buffers        *BufferPool
	SmallSlices    *ByteSlicePool // 1KB
	MediumSlices   *ByteSlicePool // 4KB
	LargeSlices    *ByteSlicePool // 16KB
	JSONBuffers    *JSONBufferPool
	Maps           *MapPool
	StringBuilders *StringBuilderPool
}{
	Buffers:        NewBufferPool(),
	SmallSlices:    NewByteSlicePool(1024),
	MediumSlices:   NewByteSlicePool(4096),
	LargeSlices:    NewByteSlicePool(16384),
	JSONBuffers:    NewJSONBufferPool(),
	Maps:           NewMapPool(),
	StringBuilders: NewStringBuilderPool(),
}
