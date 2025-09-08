package clock

import "time"

// Clock provides time operations (for testing)
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
}

// RealClock implements Clock using real time
type RealClock struct{}

// NewRealClock creates a clock that uses real time
func NewRealClock() *RealClock {
	return &RealClock{}
}

// Now returns the current time
func (c *RealClock) Now() time.Time {
	return time.Now()
}

// Since returns the time elapsed since t
func (c *RealClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

// MockClock implements Clock with fixed time for testing
type MockClock struct {
	current time.Time
}

// NewMockClock creates a clock with a fixed time
func NewMockClock(t time.Time) *MockClock {
	return &MockClock{current: t}
}

// Now returns the mocked current time
func (c *MockClock) Now() time.Time {
	return c.current
}

// Since returns the duration since t using the mocked time
func (c *MockClock) Since(t time.Time) time.Duration {
	return c.current.Sub(t)
}

// Set sets the mocked time
func (c *MockClock) Set(t time.Time) {
	c.current = t
}

// Advance advances the mocked time by the given duration
func (c *MockClock) Advance(d time.Duration) {
	c.current = c.current.Add(d)
}

// DefaultClock is the default clock instance
var DefaultClock Clock = NewRealClock()

// Now returns the current time using the default clock
func Now() time.Time {
	return DefaultClock.Now()
}

// Since returns the time elapsed since t using the default clock
func Since(t time.Time) time.Duration {
	return DefaultClock.Since(t)
}
