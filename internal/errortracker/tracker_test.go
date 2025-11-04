// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package errortracker

import (
	"errors"
	"testing"
	"time"
)

func TestTracker_RecordError(t *testing.T) {
	tests := []struct {
		name          string
		threshold     int
		errorCount    int
		expectAlert   bool
		expectAlertOn int // Which error number should trigger alert
	}{
		{
			name:          "no alert below threshold",
			threshold:     3,
			errorCount:    2,
			expectAlert:   false,
			expectAlertOn: 0,
		},
		{
			name:          "alert at threshold",
			threshold:     3,
			errorCount:    3,
			expectAlert:   true,
			expectAlertOn: 3,
		},
		{
			name:          "alert only once",
			threshold:     2,
			errorCount:    5,
			expectAlert:   true,
			expectAlertOn: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := New(tt.threshold)
			alertCount := 0

			for i := 1; i <= tt.errorCount; i++ {
				err := errors.New("test error")
				shouldAlert := tracker.RecordError(err)

				if shouldAlert {
					alertCount++
					if tt.expectAlertOn != 0 && i != tt.expectAlertOn {
						t.Errorf("expected alert on error #%d, got alert on #%d", tt.expectAlertOn, i)
					}
				}
			}

			if tt.expectAlert && alertCount == 0 {
				t.Error("expected alert but got none")
			}

			if !tt.expectAlert && alertCount > 0 {
				t.Errorf("expected no alert but got %d alerts", alertCount)
			}

			if alertCount > 1 {
				t.Errorf("expected at most 1 alert, got %d", alertCount)
			}

			// Verify consecutive errors count
			consecutiveErrors := tracker.GetConsecutiveErrors()
			if consecutiveErrors != tt.errorCount {
				t.Errorf("expected %d consecutive errors, got %d", tt.errorCount, consecutiveErrors)
			}
		})
	}
}

func TestTracker_RecordSuccess(t *testing.T) {
	tracker := New(3)

	// Record some errors
	tracker.RecordError(errors.New("error 1"))
	tracker.RecordError(errors.New("error 2"))

	if tracker.GetConsecutiveErrors() != 2 {
		t.Errorf("expected 2 consecutive errors, got %d", tracker.GetConsecutiveErrors())
	}

	// Record success - should reset and indicate recovery
	wasRecovery := tracker.RecordSuccess()
	if !wasRecovery {
		t.Error("expected wasRecovery=true after errors")
	}

	if tracker.GetConsecutiveErrors() != 0 {
		t.Errorf("expected 0 consecutive errors after success, got %d", tracker.GetConsecutiveErrors())
	}

	// Another success without previous errors should not indicate recovery
	wasRecovery = tracker.RecordSuccess()
	if wasRecovery {
		t.Error("expected wasRecovery=false when no previous errors")
	}
}

func TestTracker_GetStats(t *testing.T) {
	tracker := New(5)

	// Initial state
	consecutiveErrors, lastErr, _, _ := tracker.GetStats()
	if consecutiveErrors != 0 {
		t.Errorf("expected 0 initial errors, got %d", consecutiveErrors)
	}
	if lastErr != nil {
		t.Errorf("expected nil initial error, got %v", lastErr)
	}

	// Record an error
	testErr := errors.New("test error")
	beforeError := time.Now()
	tracker.RecordError(testErr)
	afterError := time.Now()

	var errorTime time.Time
	consecutiveErrors, lastErr, _, errorTime = tracker.GetStats()
	if consecutiveErrors != 1 {
		t.Errorf("expected 1 consecutive error, got %d", consecutiveErrors)
	}
	if lastErr.Error() != testErr.Error() {
		t.Errorf("expected error '%v', got '%v'", testErr, lastErr)
	}
	if errorTime.Before(beforeError) || errorTime.After(afterError) {
		t.Error("error time not within expected range")
	}

	// Record success
	beforeSuccess := time.Now()
	tracker.RecordSuccess()
	afterSuccess := time.Now()

	var lastSuccess time.Time
	consecutiveErrors, _, lastSuccess, _ = tracker.GetStats()
	if consecutiveErrors != 0 {
		t.Errorf("expected 0 errors after success, got %d", consecutiveErrors)
	}
	if lastSuccess.Before(beforeSuccess) || lastSuccess.After(afterSuccess) {
		t.Error("success time not within expected range")
	}
}

func TestTracker_Reset(t *testing.T) {
	tracker := New(2)

	// Record errors to trigger alert
	tracker.RecordError(errors.New("error 1"))
	tracker.RecordError(errors.New("error 2"))

	// Verify state before reset
	if tracker.GetConsecutiveErrors() != 2 {
		t.Error("expected 2 consecutive errors before reset")
	}

	// Reset
	tracker.Reset()

	// Verify state after reset
	consecutiveErrors, lastErr, _, errorTime := tracker.GetStats()
	if consecutiveErrors != 0 {
		t.Errorf("expected 0 errors after reset, got %d", consecutiveErrors)
	}
	if lastErr != nil {
		t.Errorf("expected nil error after reset, got %v", lastErr)
	}
	if !errorTime.IsZero() {
		t.Error("expected zero error time after reset")
	}

	// After reset, should be able to alert again (threshold is 2)
	tracker.RecordError(errors.New("new error 1"))
	shouldAlert := tracker.RecordError(errors.New("new error 2"))
	if !shouldAlert {
		t.Error("expected alert after reset when threshold reached again")
	}
}

func TestTracker_ThreadSafety(t *testing.T) {
	tracker := New(10)
	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				tracker.RecordError(errors.New("concurrent error"))
			}
			done <- true
		}()
	}

	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				tracker.GetConsecutiveErrors()
				_, _, _, _ = tracker.GetStats()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}

	// Just verify it didn't panic and has errors recorded
	if tracker.GetConsecutiveErrors() == 0 {
		t.Error("expected some errors to be recorded")
	}
}
