package sdk

import (
	"sync"
)

var (
	defaultClient *Client
	once          sync.Once
)

// DefaultClient returns a singleton client instance with lazy daemon autostart
func DefaultClient() *Client {
	once.Do(func() {
		c, err := NewClientFromEnv()
		if err != nil {
			// Fallback to default client creation
			c, _ = NewClient()
		}

		if c != nil {
			// Ensure daemon is running (guarded to only run once)
			if err := c.EnsureDaemonRunning(); err != nil {
				// Log warning but continue - some commands might work in read-only mode
				// In a real implementation, you might want to set a flag for read-only mode
				_ = err // Explicitly acknowledge we're ignoring the error
			}
			defaultClient = c
		}
	})

	return defaultClient
}

// NewClientFromEnv creates a client from environment variables
func NewClientFromEnv() (*Client, error) {
	// This reads TAKL_SOCKET and other environment variables
	return NewClient()
}

// ResetDefaultClient resets the singleton for testing
func ResetDefaultClient() {
	once = sync.Once{}
	defaultClient = nil
}
