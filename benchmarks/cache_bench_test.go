// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package benchmarks

import (
	"fmt"
	"testing"
	"time"

	"github.com/soothill/hyundai-logger/internal/cache"
)

// BenchmarkCacheSet benchmarks cache write operations
func BenchmarkCacheSet(b *testing.B) {
	c := cache.New(60 * time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i)
		c.Set(key, "test_value")
	}
}

// BenchmarkCacheGet benchmarks cache read operations
func BenchmarkCacheGet(b *testing.B) {
	c := cache.New(60 * time.Second)

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key_%d", i)
		c.Set(key, "test_value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i%1000)
		_ = c.Get(key)
	}
}

// BenchmarkCacheGetMiss benchmarks cache misses
func BenchmarkCacheGetMiss(b *testing.B) {
	c := cache.New(60 * time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i)
		_ = c.Get(key)
	}
}

// BenchmarkCacheGetExpired benchmarks expired entry cleanup
func BenchmarkCacheGetExpired(b *testing.B) {
	c := cache.New(1 * time.Nanosecond) // Very short TTL

	// Pre-populate cache with expired entries
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key_%d", i)
		c.Set(key, "test_value")
	}

	time.Sleep(2 * time.Millisecond) // Ensure expiration

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i%1000)
		_ = c.Get(key)
	}
}

// BenchmarkCacheSetParallel benchmarks parallel cache writes
func BenchmarkCacheSetParallel(b *testing.B) {
	c := cache.New(60 * time.Second)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key_%d", i)
			c.Set(key, "test_value")
			i++
		}
	})
}

// BenchmarkCacheGetParallel benchmarks parallel cache reads
func BenchmarkCacheGetParallel(b *testing.B) {
	c := cache.New(60 * time.Second)

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key_%d", i)
		c.Set(key, "test_value")
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key_%d", i%1000)
			_ = c.Get(key)
			i++
		}
	})
}

// BenchmarkCacheMixedOperations benchmarks realistic mixed read/write workload
func BenchmarkCacheMixedOperations(b *testing.B) {
	c := cache.New(60 * time.Second)

	// Pre-populate cache
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("key_%d", i)
		c.Set(key, "test_value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 80% reads, 20% writes (typical pattern)
		if i%5 == 0 {
			key := fmt.Sprintf("key_%d", i)
			c.Set(key, "test_value")
		} else {
			key := fmt.Sprintf("key_%d", i%500)
			_ = c.Get(key)
		}
	}
}

// BenchmarkCacheCleanExpired benchmarks expired entry cleanup
func BenchmarkCacheCleanExpired(b *testing.B) {
	c := cache.New(1 * time.Nanosecond)

	// Pre-populate with 10,000 expired entries
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key_%d", i)
		c.Set(key, "test_value")
	}

	time.Sleep(2 * time.Millisecond) // Ensure expiration

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.CleanExpired()

		// Re-populate for next iteration
		if i < b.N-1 {
			for j := 0; j < 10000; j++ {
				key := fmt.Sprintf("key_%d", j)
				c.Set(key, "test_value")
			}
			time.Sleep(2 * time.Millisecond)
		}
	}
}

// BenchmarkCacheGetStats benchmarks statistics gathering
func BenchmarkCacheGetStats(b *testing.B) {
	c := cache.New(60 * time.Second)

	// Pre-populate cache with mix of fresh and expired entries
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key_%d", i)
		if i%2 == 0 {
			c.SetWithTTL(key, "test_value", 1*time.Nanosecond) // Expired
		} else {
			c.Set(key, "test_value") // Fresh
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.GetStats()
	}
}
