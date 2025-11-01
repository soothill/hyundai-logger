// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package metrics

import (
	"fmt"
	"sync"
	"time"
)

// Collector collects application metrics for observability
type Collector struct {
	mu sync.RWMutex

	// Poll metrics
	TotalPolls         int64
	SuccessfulPolls    int64
	FailedPolls        int64
	LastPollDuration   time.Duration
	AveragePollDuration time.Duration
	totalPollDuration  time.Duration

	// Vehicle metrics
	TotalVehicles      int
	VehiclesPollSuccess map[string]int64 // VIN -> success count
	VehiclesPollFailed  map[string]int64 // VIN -> failure count

	// API metrics
	APICallsTotal      int64
	APICallsSuccess    int64
	APICallsFailed     int64
	LastAPICallTime    time.Time
	AverageAPILatency  time.Duration
	totalAPILatency    time.Duration

	// Error metrics
	ConsecutiveErrors  int64
	TotalErrors        int64
	LastError          string
	LastErrorTime      time.Time

	// Charging metrics
	ChargingDetections int64
	VehiclesCharging   int

	// Start time
	StartTime          time.Time
}

// New creates a new metrics collector
func New() *Collector {
	return &Collector{
		VehiclesPollSuccess: make(map[string]int64),
		VehiclesPollFailed:  make(map[string]int64),
		StartTime:          time.Now(),
	}
}

// RecordPollStart records the start of a poll cycle
func (c *Collector) RecordPollStart() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.TotalPolls++
}

// RecordPollComplete records completion of a poll cycle
func (c *Collector) RecordPollComplete(duration time.Duration, success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.LastPollDuration = duration
	c.totalPollDuration += duration

	if success {
		c.SuccessfulPolls++
		c.ConsecutiveErrors = 0
	} else {
		c.FailedPolls++
		c.ConsecutiveErrors++
	}

	if c.TotalPolls > 0 {
		c.AveragePollDuration = c.totalPollDuration / time.Duration(c.TotalPolls)
	}
}

// RecordVehiclePoll records a vehicle poll attempt
func (c *Collector) RecordVehiclePoll(vin string, success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if success {
		c.VehiclesPollSuccess[vin]++
	} else {
		c.VehiclesPollFailed[vin]++
	}
}

// RecordAPICall records an API call
func (c *Collector) RecordAPICall(duration time.Duration, success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.APICallsTotal++
	c.LastAPICallTime = time.Now()
	c.totalAPILatency += duration

	if success {
		c.APICallsSuccess++
	} else {
		c.APICallsFailed++
	}

	if c.APICallsTotal > 0 {
		c.AverageAPILatency = c.totalAPILatency / time.Duration(c.APICallsTotal)
	}
}

// RecordError records an error occurrence
func (c *Collector) RecordError(errorMsg string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.TotalErrors++
	c.LastError = errorMsg
	c.LastErrorTime = time.Now()
}

// RecordChargingDetection records when a vehicle is detected as charging
func (c *Collector) RecordChargingDetection() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ChargingDetections++
}

// UpdateVehiclesCharging updates the count of currently charging vehicles
func (c *Collector) UpdateVehiclesCharging(count int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.VehiclesCharging = count
}

// SetTotalVehicles sets the total number of vehicles
func (c *Collector) SetTotalVehicles(count int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.TotalVehicles = count
}

// GetStats returns a snapshot of current metrics
func (c *Collector) GetStats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Calculate success rate
	var pollSuccessRate float64
	if c.TotalPolls > 0 {
		pollSuccessRate = float64(c.SuccessfulPolls) / float64(c.TotalPolls) * 100
	}

	var apiSuccessRate float64
	if c.APICallsTotal > 0 {
		apiSuccessRate = float64(c.APICallsSuccess) / float64(c.APICallsTotal) * 100
	}

	uptime := time.Since(c.StartTime)

	return Stats{
		Uptime:              uptime,
		TotalPolls:          c.TotalPolls,
		SuccessfulPolls:     c.SuccessfulPolls,
		FailedPolls:         c.FailedPolls,
		PollSuccessRate:     pollSuccessRate,
		LastPollDuration:    c.LastPollDuration,
		AveragePollDuration: c.AveragePollDuration,
		TotalVehicles:       c.TotalVehicles,
		APICallsTotal:       c.APICallsTotal,
		APICallsSuccess:     c.APICallsSuccess,
		APICallsFailed:      c.APICallsFailed,
		APISuccessRate:      apiSuccessRate,
		AverageAPILatency:   c.AverageAPILatency,
		ConsecutiveErrors:   c.ConsecutiveErrors,
		TotalErrors:         c.TotalErrors,
		LastError:           c.LastError,
		LastErrorTime:       c.LastErrorTime,
		ChargingDetections:  c.ChargingDetections,
		VehiclesCharging:    c.VehiclesCharging,
	}
}

// Stats represents a snapshot of metrics
type Stats struct {
	Uptime              time.Duration
	TotalPolls          int64
	SuccessfulPolls     int64
	FailedPolls         int64
	PollSuccessRate     float64
	LastPollDuration    time.Duration
	AveragePollDuration time.Duration
	TotalVehicles       int
	APICallsTotal       int64
	APICallsSuccess     int64
	APICallsFailed      int64
	APISuccessRate      float64
	AverageAPILatency   time.Duration
	ConsecutiveErrors   int64
	TotalErrors         int64
	LastError           string
	LastErrorTime       time.Time
	ChargingDetections  int64
	VehiclesCharging    int
}

// FormatStats returns a human-readable string of metrics
func FormatStats(s Stats) string {
	return fmt.Sprintf(`
=== Hyundai Logger Metrics ===
Uptime: %s

Poll Metrics:
  Total Polls: %d
  Successful: %d (%.1f%%)
  Failed: %d
  Last Duration: %s
  Average Duration: %s

Vehicle Metrics:
  Total Vehicles: %d
  Currently Charging: %d
  Charging Detections: %d

API Metrics:
  Total Calls: %d
  Successful: %d (%.1f%%)
  Failed: %d
  Average Latency: %s

Error Metrics:
  Total Errors: %d
  Consecutive Errors: %d
  Last Error: %s
  Last Error Time: %s
=============================
`,
		s.Uptime.Round(time.Second),
		s.TotalPolls,
		s.SuccessfulPolls, s.PollSuccessRate,
		s.FailedPolls,
		s.LastPollDuration.Round(time.Millisecond),
		s.AveragePollDuration.Round(time.Millisecond),
		s.TotalVehicles,
		s.VehiclesCharging,
		s.ChargingDetections,
		s.APICallsTotal,
		s.APICallsSuccess, s.APISuccessRate,
		s.APICallsFailed,
		s.AverageAPILatency.Round(time.Millisecond),
		s.TotalErrors,
		s.ConsecutiveErrors,
		s.LastError,
		s.LastErrorTime.Format("2006-01-02 15:04:05"),
	)
}
