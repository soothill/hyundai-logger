// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package errortracker

import (
	"errors"
	"testing"
)

func TestTrackerRecordError(t *testing.T) {
	tracker := New(3)

	testErr := errors.New("test error")

	// Record errors below threshold
	shouldAlert := tracker.RecordError(testErr)
	if shouldAlert {
		t.Error("Should not alert below threshold")
	}

	shouldAlert = tracker.RecordError(testErr)
	if shouldAlert {
		t.Error("Should not alert below threshold")
	}

	// Third error should trigger alert
	shouldAlert = tracker.RecordError(testErr)
	if !shouldAlert {
		t.Error("Should alert at threshold")
	}

	// Fourth error should NOT alert again (hasAlerted flag prevents it)
	shouldAlert = tracker.RecordError(testErr)
	if shouldAlert {
		t.Error("Should not alert again after hasAlerted is set")
	}
}

func TestTrackerRecordSuccess(t *testing.T) {
	tracker := New(3)

	testErr := errors.New("test error")

	// Record some errors
	tracker.RecordError(testErr)
	tracker.RecordError(testErr)
	tracker.RecordError(testErr)

	// Record success - should indicate recovery
	wasRecovery := tracker.RecordSuccess()
	if !wasRecovery {
		t.Error("Should indicate recovery after errors")
	}

	// Another success should not indicate recovery
	wasRecovery = tracker.RecordSuccess()
	if wasRecovery {
		t.Error("Should not indicate recovery when already recovered")
	}
}

func TestTrackerGetStats(t *testing.T) {
	tracker := New(3)

	testErr := errors.New("test error")

	// Initial stats
	consecutive, lastErr, _, _ := tracker.GetStats()
	if consecutive != 0 {
		t.Errorf("Expected zero consecutive errors, got %d", consecutive)
	}
	if lastErr != nil {
		t.Error("Expected nil last error initially")
	}

	// Record errors
	tracker.RecordError(testErr)
	tracker.RecordError(testErr)

	consecutive, lastErr, _, errorTime := tracker.GetStats()
	if consecutive != 2 {
		t.Errorf("Expected 2 consecutive errors, got %d", consecutive)
	}
	if lastErr == nil {
		t.Error("Expected non-nil last error")
	}
	if errorTime.IsZero() {
		t.Error("Last error time should not be zero")
	}

	// Record success
	tracker.RecordSuccess()

	consecutive, _, lastSuccess, _ := tracker.GetStats()
	if consecutive != 0 {
		t.Errorf("Expected 0 consecutive errors after success, got %d", consecutive)
	}
	if lastSuccess.IsZero() {
		t.Error("Last success time should not be zero after recording success")
	}
}

func TestTrackerReset(t *testing.T) {
	tracker := New(3)

	testErr := errors.New("test error")

	// Record some errors
	tracker.RecordError(testErr)
	tracker.RecordError(testErr)

	// Reset tracker
	tracker.Reset()

	consecutive, lastErr, _, _ := tracker.GetStats()
	if consecutive != 0 {
		t.Errorf("Expected zero consecutive errors after reset, got %d", consecutive)
	}
	if lastErr != nil {
		t.Error("Expected nil last error after reset")
	}
}

func TestTrackerConcurrency(t *testing.T) {
	tracker := New(10)

	testErr := errors.New("test error")

	// Run concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			tracker.RecordError(testErr)
			tracker.RecordSuccess()
			tracker.GetStats()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic - just verify we can get stats
	_, _, _, _ = tracker.GetStats()
}

func TestTrackerThreshold(t *testing.T) {
	thresholds := []int{1, 5, 10}

	for _, threshold := range thresholds {
		tracker := New(threshold)
		testErr := errors.New("test error")

		// Record errors up to threshold - 1
		for i := 0; i < threshold-1; i++ {
			shouldAlert := tracker.RecordError(testErr)
			if shouldAlert {
				t.Errorf("Threshold %d: Should not alert at error %d", threshold, i+1)
			}
		}

		// One more error should trigger alert
		shouldAlert := tracker.RecordError(testErr)
		if !shouldAlert {
			t.Errorf("Threshold %d: Should alert at threshold", threshold)
		}
	}
}

func TestTrackerNilError(t *testing.T) {
	tracker := New(3)

	// Recording nil error still increments counter (implementation behavior)
	shouldAlert := tracker.RecordError(nil)
	if shouldAlert {
		t.Error("Should not alert on first nil error")
	}

	consecutive, lastErr, _, _ := tracker.GetStats()
	if consecutive != 1 {
		t.Errorf("Expected 1 consecutive error after nil error, got %d", consecutive)
	}
	if lastErr != nil {
		t.Error("Expected nil last error value")
	}
}
