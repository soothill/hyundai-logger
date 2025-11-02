// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package coalesce

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBasicCoalescing(t *testing.T) {
	callCount := int64(0)

	requestFn := func(ctx context.Context, key string) (interface{}, error) {
		atomic.AddInt64(&callCount, 1)
		time.Sleep(50 * time.Millisecond)
		return "result-" + key, nil
	}

	coalescer := New(Config{
		RequestFn: requestFn,
		Timeout:   1 * time.Second,
	})

	// Launch 10 concurrent requests for the same key
	var wg sync.WaitGroup
	results := make([]interface{}, 10)
	errors := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			result, err := coalescer.Do(context.Background(), "test-key")
			results[idx] = result
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// Should only call requestFn once
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}

	// All results should be the same
	for i, result := range results {
		if result != "result-test-key" {
			t.Errorf("result[%d] = %v, expected 'result-test-key'", i, result)
		}
		if errors[i] != nil {
			t.Errorf("errors[%d] = %v, expected nil", i, errors[i])
		}
	}
}

func TestDifferentKeys(t *testing.T) {
	callCount := int64(0)

	requestFn := func(ctx context.Context, key string) (interface{}, error) {
		atomic.AddInt64(&callCount, 1)
		return "result-" + key, nil
	}

	coalescer := New(Config{RequestFn: requestFn})

	// Request different keys
	result1, _ := coalescer.Do(context.Background(), "key1")
	result2, _ := coalescer.Do(context.Background(), "key2")
	result3, _ := coalescer.Do(context.Background(), "key3")

	// Should call requestFn 3 times (one per unique key)
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}

	if result1 != "result-key1" {
		t.Errorf("expected 'result-key1', got %v", result1)
	}
	if result2 != "result-key2" {
		t.Errorf("expected 'result-key2', got %v", result2)
	}
	if result3 != "result-key3" {
		t.Errorf("expected 'result-key3', got %v", result3)
	}
}

func TestTimeout(t *testing.T) {
	requestFn := func(ctx context.Context, key string) (interface{}, error) {
		time.Sleep(2 * time.Second) // Longer than timeout
		return "result", nil
	}

	coalescer := New(Config{
		RequestFn: requestFn,
		Timeout:   100 * time.Millisecond,
	})

	_, err := coalescer.Do(context.Background(), "timeout-key")

	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestContextCancellation(t *testing.T) {
	requestFn := func(ctx context.Context, key string) (interface{}, error) {
		time.Sleep(1 * time.Second)
		return "result", nil
	}

	coalescer := New(Config{RequestFn: requestFn})

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	_, err := coalescer.Do(ctx, "cancel-key")

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestErrorPropagation(t *testing.T) {
	expectedErr := errors.New("request failed")

	requestFn := func(ctx context.Context, key string) (interface{}, error) {
		return nil, expectedErr
	}

	coalescer := New(Config{RequestFn: requestFn})

	// Launch multiple requests
	var wg sync.WaitGroup
	errors := make([]error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := coalescer.Do(context.Background(), "error-key")
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// All should get the same error
	for i, err := range errors {
		if err != expectedErr {
			t.Errorf("errors[%d] = %v, expected %v", i, err, expectedErr)
		}
	}
}

func TestInFlight(t *testing.T) {
	requestFn := func(ctx context.Context, key string) (interface{}, error) {
		time.Sleep(100 * time.Millisecond)
		return "result", nil
	}

	coalescer := New(Config{RequestFn: requestFn})

	// Check initial state
	if count := coalescer.InFlight(); count != 0 {
		t.Errorf("expected 0 in-flight, got %d", count)
	}

	// Start requests
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			coalescer.Do(context.Background(), "inflight-key")
			done <- true
		}()
	}

	// Wait a bit for requests to start
	time.Sleep(10 * time.Millisecond)

	// Should have 1 in-flight (coalesced)
	if count := coalescer.InFlight(); count != 1 {
		t.Errorf("expected 1 in-flight, got %d", count)
	}

	// Wait for completion
	for i := 0; i < 5; i++ {
		<-done
	}

	// Should be back to 0
	time.Sleep(10 * time.Millisecond)
	if count := coalescer.InFlight(); count != 0 {
		t.Errorf("expected 0 in-flight after completion, got %d", count)
	}
}

func TestClear(t *testing.T) {
	requestFn := func(ctx context.Context, key string) (interface{}, error) {
		time.Sleep(100 * time.Millisecond)
		return "result", nil
	}

	coalescer := New(Config{RequestFn: requestFn})

	// Start request
	go coalescer.Do(context.Background(), "clear-key")

	// Wait for it to start
	time.Sleep(10 * time.Millisecond)

	// Clear (waits for completion)
	coalescer.Clear()

	// Should be empty
	if count := coalescer.InFlight(); count != 0 {
		t.Errorf("expected 0 in-flight after clear, got %d", count)
	}
}

func TestVehicleCoalescer(t *testing.T) {
	callCount := int64(0)

	requestFn := func(ctx context.Context, key string) (interface{}, error) {
		atomic.AddInt64(&callCount, 1)
		time.Sleep(10 * time.Millisecond)
		return map[string]interface{}{"vin": key, "status": "ok"}, nil
	}

	vc := NewVehicleCoalescer(requestFn, 1*time.Second)

	// Launch concurrent requests for same vehicle
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = vc.Get(context.Background(), "VIN123")
		}()
	}

	wg.Wait()

	// Should only call once (coalesced)
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}

	// Check stats
	stats := vc.GetStats()
	if stats.TotalRequests != 5 {
		t.Errorf("expected 5 total requests, got %d", stats.TotalRequests)
	}
	if stats.UniqueRequests != 1 {
		t.Errorf("expected 1 unique request, got %d", stats.UniqueRequests)
	}
	if stats.CoalescedHits != 4 {
		t.Errorf("expected 4 coalesced hits, got %d", stats.CoalescedHits)
	}

	// Cache hit rate should be 80% (4 out of 5)
	expectedRate := 80.0
	if stats.CacheHitRate < expectedRate-1 || stats.CacheHitRate > expectedRate+1 {
		t.Errorf("expected cache hit rate ~%.1f%%, got %.1f%%", expectedRate, stats.CacheHitRate)
	}
}

func TestVehicleCoalescerStats(t *testing.T) {
	requestFn := func(ctx context.Context, key string) (interface{}, error) {
		return "result", nil
	}

	vc := NewVehicleCoalescer(requestFn, 1*time.Second)

	// Make requests
	vc.Get(context.Background(), "VIN1")
	vc.Get(context.Background(), "VIN2")
	vc.Get(context.Background(), "VIN1") // Different request (not concurrent)

	stats := vc.GetStats()

	if stats.TotalRequests != 3 {
		t.Errorf("expected 3 total requests, got %d", stats.TotalRequests)
	}
	if stats.UniqueRequests != 3 {
		t.Errorf("expected 3 unique requests, got %d", stats.UniqueRequests)
	}
	if stats.CoalescedHits != 0 {
		t.Errorf("expected 0 coalesced hits (sequential), got %d", stats.CoalescedHits)
	}

	// Reset stats
	vc.ResetStats()
	stats = vc.GetStats()

	if stats.TotalRequests != 0 {
		t.Errorf("expected 0 requests after reset, got %d", stats.TotalRequests)
	}
}

func TestBatchCoalescer(t *testing.T) {
	processed := make(map[string]bool)
	var mu sync.Mutex

	requestFn := func(ctx context.Context, key string) (interface{}, error) {
		mu.Lock()
		processed[key] = true
		mu.Unlock()
		return "result-" + key, nil
	}

	bc := NewBatchCoalescer(requestFn, 3, 100*time.Millisecond)

	// Add keys
	bc.Add("key1")
	bc.Add("key2")
	bc.Add("key3") // Should trigger flush

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if len(processed) != 3 {
		t.Errorf("expected 3 processed keys, got %d", len(processed))
	}
	mu.Unlock()
}

func TestBatchCoalescerWindow(t *testing.T) {
	processed := make(map[string]bool)
	var mu sync.Mutex

	requestFn := func(ctx context.Context, key string) (interface{}, error) {
		mu.Lock()
		processed[key] = true
		mu.Unlock()
		return "result", nil
	}

	bc := NewBatchCoalescer(requestFn, 10, 50*time.Millisecond)

	// Add keys (less than batch size)
	bc.Add("key1")
	bc.Add("key2")

	// Wait for window to expire
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if len(processed) != 2 {
		t.Errorf("expected 2 processed keys, got %d", len(processed))
	}
	mu.Unlock()
}

func TestBatchCoalescerManualFlush(t *testing.T) {
	processed := make(map[string]bool)
	var mu sync.Mutex

	requestFn := func(ctx context.Context, key string) (interface{}, error) {
		mu.Lock()
		processed[key] = true
		mu.Unlock()
		return "result", nil
	}

	bc := NewBatchCoalescer(requestFn, 100, 10*time.Second)

	// Add keys
	bc.Add("key1")
	bc.Add("key2")

	// Manual flush
	bc.Flush()

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if len(processed) != 2 {
		t.Errorf("expected 2 processed keys, got %d", len(processed))
	}
	mu.Unlock()
}

// Benchmark tests
func BenchmarkCoalescing(b *testing.B) {
	requestFn := func(ctx context.Context, key string) (interface{}, error) {
		time.Sleep(1 * time.Millisecond)
		return "result", nil
	}

	coalescer := New(Config{RequestFn: requestFn})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			coalescer.Do(context.Background(), "benchmark-key")
		}
	})
}

func BenchmarkWithoutCoalescing(b *testing.B) {
	requestFn := func(ctx context.Context, key string) (interface{}, error) {
		time.Sleep(1 * time.Millisecond)
		return "result", nil
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			requestFn(context.Background(), "benchmark-key")
		}
	})
}
