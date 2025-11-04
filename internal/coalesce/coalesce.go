// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package coalesce

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Result represents the result of a coalesced request
type Result struct {
	Value interface{}
	Err   error
}

// RequestFunc is a function that performs the actual request
type RequestFunc func(ctx context.Context, key string) (interface{}, error)

// inflight represents an in-flight request
type inflight struct {
	wg     sync.WaitGroup
	result *Result
	done   chan struct{}
}

// Coalescer deduplicates identical in-flight requests
type Coalescer struct {
	mu        sync.Mutex
	requests  map[string]*inflight
	timeout   time.Duration
	requestFn RequestFunc
}

// Config configures request coalescing
type Config struct {
	Timeout   time.Duration // Maximum time to wait for a request
	RequestFn RequestFunc   // Function to execute for requests
}

// New creates a new request coalescer
func New(config Config) *Coalescer {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Coalescer{
		requests:  make(map[string]*inflight),
		timeout:   config.Timeout,
		requestFn: config.RequestFn,
	}
}

// Do executes a request, coalescing duplicate requests
func (c *Coalescer) Do(ctx context.Context, key string) (interface{}, error) {
	c.mu.Lock()

	// Check if request is already in flight
	if req, exists := c.requests[key]; exists {
		c.mu.Unlock()

		// Wait for existing request to complete
		select {
		case <-req.done:
			return req.result.Value, req.result.Err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Create new in-flight request
	req := &inflight{
		done: make(chan struct{}),
	}
	req.wg.Add(1)
	c.requests[key] = req
	c.mu.Unlock()

	// Execute request
	value, err := c.executeRequest(ctx, key)

	// Store result
	req.result = &Result{
		Value: value,
		Err:   err,
	}

	// Mark as done
	close(req.done)
	req.wg.Done()

	// Remove from in-flight map
	c.mu.Lock()
	delete(c.requests, key)
	c.mu.Unlock()

	return value, err
}

// executeRequest performs the actual request with timeout
func (c *Coalescer) executeRequest(ctx context.Context, key string) (interface{}, error) {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Execute request
	resultChan := make(chan Result, 1)

	go func() {
		value, err := c.requestFn(timeoutCtx, key)
		resultChan <- Result{Value: value, Err: err}
	}()

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		return result.Value, result.Err
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("request timeout: %w", timeoutCtx.Err())
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// InFlight returns the number of in-flight requests
func (c *Coalescer) InFlight() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.requests)
}

// Clear removes all in-flight requests (for cleanup)
func (c *Coalescer) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Wait for all requests to complete
	for _, req := range c.requests {
		req.wg.Wait()
	}

	// Clear map
	c.requests = make(map[string]*inflight)
}

// VehicleCoalescer is specialized for vehicle status requests
type VehicleCoalescer struct {
	*Coalescer
	stats Stats
	mu    sync.Mutex
}

// Stats tracks coalescing statistics
type Stats struct {
	TotalRequests  int64
	CoalescedHits  int64
	UniqueRequests int64
	TimeoutErrors  int64
	CacheHitRate   float64
}

// NewVehicleCoalescer creates a coalescer for vehicle requests
func NewVehicleCoalescer(requestFn RequestFunc, timeout time.Duration) *VehicleCoalescer {
	return &VehicleCoalescer{
		Coalescer: New(Config{
			Timeout:   timeout,
			RequestFn: requestFn,
		}),
	}
}

// Get fetches vehicle data, coalescing duplicate requests
func (vc *VehicleCoalescer) Get(ctx context.Context, vehicleID string) (interface{}, error) {
	vc.mu.Lock()
	vc.stats.TotalRequests++

	// Check if request is in flight
	vc.Coalescer.mu.Lock()
	if _, exists := vc.Coalescer.requests[vehicleID]; exists {
		vc.stats.CoalescedHits++
	} else {
		vc.stats.UniqueRequests++
	}
	vc.Coalescer.mu.Unlock()

	// Update cache hit rate
	if vc.stats.TotalRequests > 0 {
		vc.stats.CacheHitRate = float64(vc.stats.CoalescedHits) / float64(vc.stats.TotalRequests) * 100.0
	}

	vc.mu.Unlock()

	// Execute request
	value, err := vc.Do(ctx, vehicleID)

	// Track timeouts
	if err != nil && (err == context.DeadlineExceeded || err == context.Canceled) {
		vc.mu.Lock()
		vc.stats.TimeoutErrors++
		vc.mu.Unlock()
	}

	return value, err
}

// GetStats returns current statistics
func (vc *VehicleCoalescer) GetStats() Stats {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	return vc.stats
}

// ResetStats resets statistics
func (vc *VehicleCoalescer) ResetStats() {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.stats = Stats{}
}

// BatchCoalescer handles batch requests with coalescing
type BatchCoalescer struct {
	coalescer *Coalescer
	batchSize int
	window    time.Duration
	mu        sync.Mutex
	batch     []string
	timer     *time.Timer
}

// NewBatchCoalescer creates a batch coalescer
func NewBatchCoalescer(requestFn RequestFunc, batchSize int, window time.Duration) *BatchCoalescer {
	return &BatchCoalescer{
		coalescer: New(Config{RequestFn: requestFn}),
		batchSize: batchSize,
		window:    window,
		batch:     make([]string, 0, batchSize),
	}
}

// Add adds a key to the batch
func (bc *BatchCoalescer) Add(key string) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.batch = append(bc.batch, key)

	// Flush if batch is full
	if len(bc.batch) >= bc.batchSize {
		bc.flush()
		return
	}

	// Set timer for window
	// Stop any existing timer to prevent leaks before creating a new one
	if bc.timer != nil {
		bc.timer.Stop()
	}
	bc.timer = time.AfterFunc(bc.window, func() {
		bc.mu.Lock()
		bc.flush()
		bc.mu.Unlock()
	})
}

// flush processes the current batch
func (bc *BatchCoalescer) flush() {
	if len(bc.batch) == 0 {
		return
	}

	// Process batch
	for _, key := range bc.batch {
		go func(k string) {
			_, _ = bc.coalescer.Do(context.Background(), k)
		}(key)
	}

	// Reset batch
	bc.batch = bc.batch[:0]

	// Stop timer
	if bc.timer != nil {
		bc.timer.Stop()
		bc.timer = nil
	}
}

// Flush manually flushes the batch
func (bc *BatchCoalescer) Flush() {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.flush()
}
