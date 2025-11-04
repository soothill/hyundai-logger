// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package cache

import (
	"sync"
	"time"
)

// Entry represents a cached entry
type Entry struct {
	Data      interface{}
	Timestamp time.Time
	TTL       time.Duration
}

// IsExpired checks if the cache entry has expired
func (e *Entry) IsExpired() bool {
	return time.Since(e.Timestamp) > e.TTL
}

// Cache provides a simple in-memory cache with TTL support
type Cache struct {
	mu          sync.RWMutex
	entries     map[string]*Entry
	defaultTTL  time.Duration
	stopCleanup chan struct{}
}

// New creates a new cache with the specified default TTL
// Automatically starts a background cleanup task
func New(defaultTTL time.Duration) *Cache {
	c := &Cache{
		entries:    make(map[string]*Entry),
		defaultTTL: defaultTTL,
	}

	// Auto-start cleanup task with 1/4 of TTL interval
	cleanupInterval := defaultTTL / 4
	if cleanupInterval < time.Second {
		cleanupInterval = time.Second
	}
	c.stopCleanup = c.StartCleanupTask(cleanupInterval)

	return c
}

// Get retrieves a value from the cache
// Returns nil if the key doesn't exist or has expired
// Expired entries are cleaned up by the background cleanup task
func (c *Cache) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists || entry.IsExpired() {
		return nil
	}

	return entry.Data
}

// Set stores a value in the cache with the default TTL
func (c *Cache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL stores a value in the cache with a custom TTL
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &Entry{
		Data:      value,
		Timestamp: time.Now(),
		TTL:       ttl,
	}
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*Entry)
}

// Close stops the background cleanup task
// Should be called when the cache is no longer needed
func (c *Cache) Close() {
	if c.stopCleanup != nil {
		close(c.stopCleanup)
	}
}

// Size returns the number of entries in the cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}

// CleanExpired removes all expired entries from the cache
func (c *Cache) CleanExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	for key, entry := range c.entries {
		if entry.IsExpired() {
			delete(c.entries, key)
			removed++
		}
	}

	return removed
}

// StartCleanupTask starts a background goroutine that periodically cleans up expired entries
func (c *Cache) StartCleanupTask(interval time.Duration) chan struct{} {
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.CleanExpired()
			case <-stop:
				return
			}
		}
	}()

	return stop
}

// GetStats returns cache statistics
func (c *Cache) GetStats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var totalSize int64
	expired := 0
	fresh := 0

	for _, entry := range c.entries {
		totalSize++
		if entry.IsExpired() {
			expired++
		} else {
			fresh++
		}
	}

	return Stats{
		TotalEntries:   int(totalSize),
		FreshEntries:   fresh,
		ExpiredEntries: expired,
	}
}

// Stats represents cache statistics
type Stats struct {
	TotalEntries   int
	FreshEntries   int
	ExpiredEntries int
}
