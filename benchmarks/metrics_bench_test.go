// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package benchmarks

import (
	"testing"
	"time"

	"github.com/soothill/hyundai-logger/internal/metrics"
)

// BenchmarkMetricsCollectorRecordPoll benchmarks poll recording
func BenchmarkMetricsCollectorRecordPoll(b *testing.B) {
	collector := metrics.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordPollComplete(100*time.Millisecond, true)
	}
}

// BenchmarkMetricsCollectorRecordError benchmarks error recording
func BenchmarkMetricsCollectorRecordError(b *testing.B) {
	collector := metrics.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordError("test error")
	}
}

// BenchmarkMetricsCollectorRecordAPICall benchmarks API call recording
func BenchmarkMetricsCollectorRecordAPICall(b *testing.B) {
	collector := metrics.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordAPICall(50*time.Millisecond, true)
	}
}

// BenchmarkMetricsCollectorRecordAPICallError benchmarks API call error recording
func BenchmarkMetricsCollectorRecordAPICallError(b *testing.B) {
	collector := metrics.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordAPICall(50*time.Millisecond, false)
	}
}

// BenchmarkMetricsCollectorUpdateVehiclesCharging benchmarks vehicle charging count updates
func BenchmarkMetricsCollectorUpdateVehiclesCharging(b *testing.B) {
	collector := metrics.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.UpdateVehiclesCharging(2)
	}
}

// BenchmarkMetricsCollectorRecordChargingDetection benchmarks charging detection recording
func BenchmarkMetricsCollectorRecordChargingDetection(b *testing.B) {
	collector := metrics.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordChargingDetection()
	}
}

// BenchmarkMetricsCollectorGetStats benchmarks statistics retrieval
func BenchmarkMetricsCollectorGetStats(b *testing.B) {
	collector := metrics.New()

	// Pre-populate with some data
	for i := 0; i < 100; i++ {
		collector.RecordPollComplete(100*time.Millisecond, true)
		collector.RecordAPICall(50*time.Millisecond, true)
		if i%10 == 0 {
			collector.RecordError("test error")
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = collector.GetStats()
	}
}

// BenchmarkMetricsCollectorParallelRecording benchmarks parallel metric recording
func BenchmarkMetricsCollectorParallelRecording(b *testing.B) {
	collector := metrics.New()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 5 {
			case 0:
				collector.RecordPollComplete(100*time.Millisecond, true)
			case 1:
				collector.RecordAPICall(50*time.Millisecond, true)
			case 2:
				collector.RecordError("test error")
			case 3:
				collector.UpdateVehiclesCharging(2)
			case 4:
				collector.RecordChargingDetection()
			}
			i++
		}
	})
}

// BenchmarkMetricsCollectorMixedOperations benchmarks realistic mixed workload
func BenchmarkMetricsCollectorMixedOperations(b *testing.B) {
	collector := metrics.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate a typical poll cycle
		collector.RecordPollComplete(100*time.Millisecond, true)

		// 3 API calls per poll
		for j := 0; j < 3; j++ {
			collector.RecordAPICall(50*time.Millisecond, true)
		}

		// Occasional error (10% of polls)
		if i%10 == 0 {
			collector.RecordError("occasional error")
		}

		// Update vehicle status
		collector.UpdateVehiclesCharging(1)
		collector.SetTotalVehicles(5)

		// Occasional charging detection
		if i%5 == 0 {
			collector.RecordChargingDetection()
		}

		// Get stats every 20 iterations (monitoring)
		if i%20 == 0 {
			_ = collector.GetStats()
		}
	}
}

// BenchmarkMetricsCalculation benchmarks statistics calculation overhead
func BenchmarkMetricsCalculation(b *testing.B) {
	collector := metrics.New()

	// Populate with significant amount of data
	for i := 0; i < 10000; i++ {
		collector.RecordPollComplete(100*time.Millisecond, true)
		collector.RecordAPICall(50*time.Millisecond, true)
		if i%100 == 0 {
			collector.RecordError("error")
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stats := collector.GetStats()
		// Force calculation of derived metrics
		_ = stats.PollSuccessRate
		_ = stats.APISuccessRate
		_ = stats.AveragePollDuration
		_ = stats.AverageAPILatency
	}
}

// BenchmarkMetricsCollectorReset benchmarks collector reset operation
func BenchmarkMetricsCollectorReset(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector := metrics.New()

		// Populate with data
		for j := 0; j < 100; j++ {
			collector.RecordPollComplete(100*time.Millisecond, true)
			collector.RecordAPICall(50*time.Millisecond, true)
		}

		// The reset happens implicitly when we create a new collector
		_ = collector
	}
}
