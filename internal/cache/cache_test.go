// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package cache

import (
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	cache := New(1 * time.Second)
	if cache == nil {
		t.Fatal("Expected cache to be created, got nil")
	}
	if cache.defaultTTL != 1*time.Second {
		t.Errorf("Expected default TTL of 1s, got %v", cache.defaultTTL)
	}
}

func TestSetAndGet(t *testing.T) {
	cache := New(1 * time.Second)

	// Set a value
	cache.Set("key1", "value1")

	// Get the value
	value := cache.Get("key1")
	if value != "value1" {
		t.Errorf("Expected 'value1', got %v", value)
	}

	// Get non-existent key
	value = cache.Get("nonexistent")
	if value != nil {
		t.Errorf("Expected nil for non-existent key, got %v", value)
	}
}

func TestSetWithTTL(t *testing.T) {
	cache := New(1 * time.Hour)

	// Set with short TTL
	cache.SetWithTTL("short", "value", 100*time.Millisecond)

	// Get immediately
	value := cache.Get("short")
	if value != "value" {
		t.Errorf("Expected 'value', got %v", value)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	value = cache.Get("short")
	if value != nil {
		t.Errorf("Expected nil for expired key, got %v", value)
	}
}

func TestExpiration(t *testing.T) {
	cache := New(50 * time.Millisecond)

	cache.Set("key1", "value1")

	// Get before expiration
	value := cache.Get("key1")
	if value != "value1" {
		t.Errorf("Expected 'value1', got %v", value)
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	value = cache.Get("key1")
	if value != nil {
		t.Errorf("Expected nil for expired key, got %v", value)
	}

	// Verify entry is still present but expired (cleaned by background task)
	// Manually trigger cleanup to verify deletion
	removed := cache.CleanExpired()
	if removed != 1 {
		t.Errorf("Expected 1 expired entry to be removed, got %d", removed)
	}

	// Now verify entry was deleted
	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0 after expiration cleanup, got %d", cache.Size())
	}
}

func TestDelete(t *testing.T) {
	cache := New(1 * time.Second)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Delete one key
	cache.Delete("key1")

	// Verify deletion
	if cache.Get("key1") != nil {
		t.Error("Expected key1 to be deleted")
	}
	if cache.Get("key2") != "value2" {
		t.Error("Expected key2 to still exist")
	}
}

func TestClear(t *testing.T) {
	cache := New(1 * time.Second)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	if cache.Size() != 3 {
		t.Errorf("Expected size 3, got %d", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", cache.Size())
	}
}

func TestSize(t *testing.T) {
	cache := New(1 * time.Second)

	if cache.Size() != 0 {
		t.Errorf("Expected initial size 0, got %d", cache.Size())
	}

	cache.Set("key1", "value1")
	if cache.Size() != 1 {
		t.Errorf("Expected size 1, got %d", cache.Size())
	}

	cache.Set("key2", "value2")
	cache.Set("key3", "value3")
	if cache.Size() != 3 {
		t.Errorf("Expected size 3, got %d", cache.Size())
	}

	cache.Delete("key1")
	if cache.Size() != 2 {
		t.Errorf("Expected size 2 after delete, got %d", cache.Size())
	}
}

func TestCleanExpired(t *testing.T) {
	cache := New(50 * time.Millisecond)

	// Add entries
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.SetWithTTL("key3", "value3", 1*time.Hour) // Won't expire

	// Wait for some to expire
	time.Sleep(100 * time.Millisecond)

	// Clean expired
	removed := cache.CleanExpired()

	if removed != 2 {
		t.Errorf("Expected 2 expired entries, got %d", removed)
	}

	// Verify only non-expired remains
	if cache.Size() != 1 {
		t.Errorf("Expected size 1 after cleanup, got %d", cache.Size())
	}

	if cache.Get("key3") != "value3" {
		t.Error("Expected key3 to still exist")
	}
}

func TestStartCleanupTask(t *testing.T) {
	cache := New(50 * time.Millisecond)

	// Add entries
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Start cleanup task
	stop := cache.StartCleanupTask(30 * time.Millisecond)
	defer close(stop)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Wait for cleanup to run
	time.Sleep(50 * time.Millisecond)

	// Should be cleaned up
	if cache.Size() != 0 {
		t.Errorf("Expected size 0 after automatic cleanup, got %d", cache.Size())
	}
}

func TestGetStats(t *testing.T) {
	cache := New(100 * time.Millisecond)

	// Add fresh entries
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Add entry that will expire
	cache.SetWithTTL("expired", "value", 1*time.Millisecond)
	time.Sleep(10 * time.Millisecond) // Wait for it to expire

	stats := cache.GetStats()

	if stats.TotalEntries != 3 {
		t.Errorf("Expected 3 total entries, got %d", stats.TotalEntries)
	}
	if stats.FreshEntries != 2 {
		t.Errorf("Expected 2 fresh entries, got %d", stats.FreshEntries)
	}
	if stats.ExpiredEntries != 1 {
		t.Errorf("Expected 1 expired entry, got %d", stats.ExpiredEntries)
	}
}

func TestConcurrentAccess(t *testing.T) {
	cache := New(1 * time.Second)
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := string(rune('A' + idx%26))
			cache.Set(key, idx)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := string(rune('A' + idx%26))
			cache.Get(key)
		}(i)
	}

	wg.Wait()

	// Verify no panic occurred
	if cache.Size() > 26 {
		t.Errorf("Expected at most 26 entries (A-Z), got %d", cache.Size())
	}
}

func TestDifferentDataTypes(t *testing.T) {
	cache := New(1 * time.Second)

	// Store different types
	cache.Set("string", "hello")
	cache.Set("int", 42)
	cache.Set("bool", true)
	cache.Set("struct", struct{ Name string }{Name: "test"})
	cache.Set("slice", []int{1, 2, 3})
	cache.Set("map", map[string]int{"a": 1})

	// Retrieve and verify types
	if v, ok := cache.Get("string").(string); !ok || v != "hello" {
		t.Error("String value mismatch")
	}
	if v, ok := cache.Get("int").(int); !ok || v != 42 {
		t.Error("Int value mismatch")
	}
	if v, ok := cache.Get("bool").(bool); !ok || v != true {
		t.Error("Bool value mismatch")
	}
}

func TestOverwrite(t *testing.T) {
	cache := New(1 * time.Second)

	cache.Set("key", "value1")
	if cache.Get("key") != "value1" {
		t.Error("Expected value1")
	}

	// Overwrite
	cache.Set("key", "value2")
	if cache.Get("key") != "value2" {
		t.Error("Expected value2 after overwrite")
	}
}

func TestLongRunningCleanup(t *testing.T) {
	cache := New(20 * time.Millisecond)

	// Start cleanup task
	stop := cache.StartCleanupTask(10 * time.Millisecond)

	// Add entries continuously
	for i := 0; i < 10; i++ {
		cache.Set(string(rune('A'+i)), i)
		time.Sleep(5 * time.Millisecond)
	}

	// Stop cleanup
	close(stop)

	// Give cleanup time to finish
	time.Sleep(50 * time.Millisecond)

	// Most entries should have been cleaned up
	if cache.Size() > 3 {
		t.Logf("Warning: Expected most entries cleaned up, got %d remaining", cache.Size())
	}
}

func TestEntryIsExpired(t *testing.T) {
	entry := &Entry{
		Data:      "test",
		Timestamp: time.Now().Add(-2 * time.Second),
		TTL:       1 * time.Second,
	}

	if !entry.IsExpired() {
		t.Error("Expected entry to be expired")
	}

	freshEntry := &Entry{
		Data:      "test",
		Timestamp: time.Now(),
		TTL:       1 * time.Hour,
	}

	if freshEntry.IsExpired() {
		t.Error("Expected entry to not be expired")
	}
}

func BenchmarkSet(b *testing.B) {
	cache := New(1 * time.Second)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("key", i)
	}
}

func BenchmarkGet(b *testing.B) {
	cache := New(1 * time.Second)
	cache.Set("key", "value")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("key")
	}
}

func BenchmarkConcurrentSetGet(b *testing.B) {
	cache := New(1 * time.Second)
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				cache.Set("key", i)
			} else {
				cache.Get("key")
			}
			i++
		}
	})
}
