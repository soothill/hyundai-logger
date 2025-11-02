// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package pool

import (
	"bytes"
	"runtime"
	"testing"
)

func TestBufferPool(t *testing.T) {
	pool := NewBufferPool()

	// Get buffer
	buf := pool.Get()
	if buf == nil {
		t.Fatal("expected buffer, got nil")
	}

	// Write some data
	buf.WriteString("test data")
	if buf.String() != "test data" {
		t.Errorf("expected 'test data', got %s", buf.String())
	}

	// Return to pool
	pool.Put(buf)

	// Get again - should be reset
	buf2 := pool.Get()
	if buf2.Len() != 0 {
		t.Errorf("buffer should be reset, got length %d", buf2.Len())
	}
}

func TestBufferPoolLargeBuffer(t *testing.T) {
	pool := NewBufferPool()

	buf := pool.Get()

	// Write 100KB of data
	largeData := make([]byte, 100*1024)
	buf.Write(largeData)

	// Should not be returned to pool (> 64KB cap)
	initialCap := buf.Cap()
	pool.Put(buf)

	// Get new buffer - should be fresh, not the large one
	buf2 := pool.Get()
	if buf2.Cap() == initialCap && initialCap > 64*1024 {
		t.Error("large buffer should not be pooled")
	}
}

func TestByteSlicePool(t *testing.T) {
	pool := NewByteSlicePool(1024)

	// Get slice
	slice := pool.Get()
	if len(slice) != 1024 {
		t.Errorf("expected slice length 1024, got %d", len(slice))
	}

	// Modify slice
	slice[0] = 42
	slice[100] = 99

	// Return to pool
	pool.Put(slice)

	// Get again
	slice2 := pool.Get()
	if len(slice2) != 1024 {
		t.Errorf("expected slice length 1024, got %d", len(slice2))
	}

	// Data might be same or different (pool behavior)
	// Just verify it's usable
	slice2[0] = 0
}

func TestJSONBufferPool(t *testing.T) {
	pool := NewJSONBufferPool()

	tests := []struct {
		name string
		size int
	}{
		{"small", 512},
		{"medium", 4096},
		{"large", 128 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := pool.Get(tt.size)
			if buf == nil {
				t.Fatal("expected buffer, got nil")
			}

			// Use buffer
			buf.WriteString("test json")

			// Return to pool
			pool.Put(buf)
		})
	}
}

func TestMapPool(t *testing.T) {
	pool := NewMapPool()

	// Get map
	m := pool.Get()
	if m == nil {
		t.Fatal("expected map, got nil")
	}

	// Add data
	m["key1"] = "value1"
	m["key2"] = 42
	m["key3"] = true

	if len(m) != 3 {
		t.Errorf("expected map length 3, got %d", len(m))
	}

	// Return to pool
	pool.Put(m)

	// Get again - should be empty
	m2 := pool.Get()
	if len(m2) != 0 {
		t.Errorf("map should be cleared, got length %d", len(m2))
	}
}

func TestMapPoolLargeMap(t *testing.T) {
	pool := NewMapPool()

	m := pool.Get()

	// Add 150 entries (> 100 limit)
	for i := 0; i < 150; i++ {
		m[string(rune(i))] = i
	}

	pool.Put(m)

	// Get new map - should be fresh
	m2 := pool.Get()
	if len(m2) != 0 {
		t.Error("should get fresh map for large map")
	}
}

func TestStringBuilderPool(t *testing.T) {
	pool := NewStringBuilderPool()

	// Get builder
	buf := pool.Get()
	if buf == nil {
		t.Fatal("expected buffer, got nil")
	}

	// Build string
	buf.WriteString("Hello, ")
	buf.WriteString("World!")

	result := buf.String()
	if result != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %s", result)
	}

	// Return to pool
	pool.Put(buf)

	// Get again - should be reset
	buf2 := pool.Get()
	if buf2.Len() != 0 {
		t.Errorf("buffer should be reset, got length %d", buf2.Len())
	}
}

func TestGlobalPools(t *testing.T) {
	// Test all global pools are initialized
	if GlobalPools.Buffers == nil {
		t.Error("GlobalPools.Buffers should not be nil")
	}
	if GlobalPools.SmallSlices == nil {
		t.Error("GlobalPools.SmallSlices should not be nil")
	}
	if GlobalPools.MediumSlices == nil {
		t.Error("GlobalPools.MediumSlices should not be nil")
	}
	if GlobalPools.LargeSlices == nil {
		t.Error("GlobalPools.LargeSlices should not be nil")
	}
	if GlobalPools.JSONBuffers == nil {
		t.Error("GlobalPools.JSONBuffers should not be nil")
	}
	if GlobalPools.Maps == nil {
		t.Error("GlobalPools.Maps should not be nil")
	}
	if GlobalPools.StringBuilders == nil {
		t.Error("GlobalPools.StringBuilders should not be nil")
	}

	// Test using global pools
	buf := GlobalPools.Buffers.Get()
	buf.WriteString("test")
	GlobalPools.Buffers.Put(buf)

	smallSlice := GlobalPools.SmallSlices.Get()
	if len(smallSlice) != 1024 {
		t.Errorf("expected small slice 1KB, got %d", len(smallSlice))
	}
	GlobalPools.SmallSlices.Put(smallSlice)

	mediumSlice := GlobalPools.MediumSlices.Get()
	if len(mediumSlice) != 4096 {
		t.Errorf("expected medium slice 4KB, got %d", len(mediumSlice))
	}
	GlobalPools.MediumSlices.Put(mediumSlice)

	largeSlice := GlobalPools.LargeSlices.Get()
	if len(largeSlice) != 16384 {
		t.Errorf("expected large slice 16KB, got %d", len(largeSlice))
	}
	GlobalPools.LargeSlices.Put(largeSlice)
}

func TestConcurrentAccess(t *testing.T) {
	pool := NewBufferPool()

	done := make(chan bool)
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				buf := pool.Get()
				buf.WriteString("test")
				pool.Put(buf)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// Benchmark tests to demonstrate GC reduction
func BenchmarkWithoutPool(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := new(bytes.Buffer)
		buf.WriteString("test data for benchmarking")
		_ = buf.String()
	}
}

func BenchmarkWithPool(b *testing.B) {
	pool := NewBufferPool()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := pool.Get()
		buf.WriteString("test data for benchmarking")
		_ = buf.String()
		pool.Put(buf)
	}
}

func BenchmarkByteSliceWithoutPool(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		slice := make([]byte, 1024)
		slice[0] = byte(i)
	}
}

func BenchmarkByteSliceWithPool(b *testing.B) {
	pool := NewByteSlicePool(1024)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slice := pool.Get()
		slice[0] = byte(i)
		pool.Put(slice)
	}
}

func BenchmarkMapWithoutPool(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m := make(map[string]interface{}, 10)
		m["key1"] = "value1"
		m["key2"] = 42
	}
}

func BenchmarkMapWithPool(b *testing.B) {
	pool := NewMapPool()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := pool.Get()
		m["key1"] = "value1"
		m["key2"] = 42
		pool.Put(m)
	}
}

func BenchmarkGCPressure(b *testing.B) {
	// Benchmark to show GC impact
	b.Run("without_pool", func(b *testing.B) {
		var ms1, ms2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&ms1)

		for i := 0; i < b.N; i++ {
			buf := new(bytes.Buffer)
			buf.Write(make([]byte, 1024))
		}

		runtime.ReadMemStats(&ms2)
		b.ReportMetric(float64(ms2.NumGC-ms1.NumGC), "GC_runs")
	})

	b.Run("with_pool", func(b *testing.B) {
		pool := NewBufferPool()
		var ms1, ms2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&ms1)

		for i := 0; i < b.N; i++ {
			buf := pool.Get()
			buf.Write(make([]byte, 1024))
			pool.Put(buf)
		}

		runtime.ReadMemStats(&ms2)
		b.ReportMetric(float64(ms2.NumGC-ms1.NumGC), "GC_runs")
	})
}

func BenchmarkJSONBufferPool(b *testing.B) {
	pool := NewJSONBufferPool()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := pool.Get(512)
		buf.WriteString(`{"key": "value"}`)
		pool.Put(buf)
	}
}
