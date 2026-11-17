package coreSystem

import "time"

// Clock provides abstraction for time operations
type Clock interface {
	Now() time.Time
	Since(time.Time) time.Duration
	Sleep(duration time.Duration)
}

// RealClock uses actual system time
type RealClock struct{}

func (rc *RealClock) Now() time.Time {
	return time.Now()
}
func (rc *RealClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func (rc *RealClock) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

// MockClock allows manual time control for testing
type MockClock struct {
	currentTime time.Time
}

func NewMockClock(startTime time.Time) *MockClock {
	return &MockClock{currentTime: startTime}
}

func (mc *MockClock) Now() time.Time {
	return mc.currentTime
}

func (mc *MockClock) Sleep(duration time.Duration) {
	// No actual sleep - just advance time
	mc.currentTime = mc.currentTime.Add(duration)
}
func (mc *MockClock) Since(t time.Time) time.Duration {
	return mc.currentTime.Sub(t)
}

// Advance manually advances mock time by the given duration
func (mc *MockClock) Advance(duration time.Duration) {
	mc.currentTime = mc.currentTime.Add(duration)
}

// Set manually sets mock time
func (mc *MockClock) Set(t time.Time) {
	mc.currentTime = t
}
