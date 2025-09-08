package shared

import "time"

// DefaultClock is the default implementation using system time
type DefaultClock struct{}

func (c DefaultClock) Now() time.Time {
	return time.Now()
}

// FixedClock always returns the same time (useful for testing)
type FixedClock struct {
	Time time.Time
}

func (c FixedClock) Now() time.Time {
	return c.Time
}
