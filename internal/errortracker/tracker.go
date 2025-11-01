// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package errortracker

import (
	"sync"
	"time"
)

// Tracker tracks consecutive errors for alerting purposes
type Tracker struct {
	mu                sync.RWMutex
	consecutiveErrors int
	lastError         error
	lastErrorTime     time.Time
	lastSuccessTime   time.Time
	threshold         int
	hasAlerted        bool
}

// New creates a new error tracker with the specified alert threshold
func New(threshold int) *Tracker {
	return &Tracker{
		threshold: threshold,
	}
}

// RecordError tracks an error occurrence and returns whether an alert should be sent
func (t *Tracker) RecordError(err error) (shouldAlert bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.consecutiveErrors++
	t.lastError = err
	t.lastErrorTime = time.Now()

	if t.consecutiveErrors >= t.threshold && !t.hasAlerted {
		t.hasAlerted = true
		return true
	}

	return false
}

// RecordSuccess resets error tracking and returns whether this was a recovery
func (t *Tracker) RecordSuccess() (wasRecovery bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	hadErrors := t.consecutiveErrors > 0
	t.consecutiveErrors = 0
	t.lastSuccessTime = time.Now()
	t.hasAlerted = false

	return hadErrors
}

// GetStats returns current error tracking statistics (thread-safe)
func (t *Tracker) GetStats() (consecutiveErrors int, lastErr error, lastSuccess time.Time, errorTime time.Time) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.consecutiveErrors, t.lastError, t.lastSuccessTime, t.lastErrorTime
}

// GetConsecutiveErrors returns the number of consecutive errors (thread-safe)
func (t *Tracker) GetConsecutiveErrors() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.consecutiveErrors
}

// Reset resets all error tracking state
func (t *Tracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.consecutiveErrors = 0
	t.lastError = nil
	t.lastErrorTime = time.Time{}
	t.hasAlerted = false
}
