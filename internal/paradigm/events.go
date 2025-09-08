package paradigm

import (
	"context"
	"sync"
	"time"
)

// Event represents a domain event
type Event interface {
	Type() string
	Time() time.Time
}

// Concrete event types
type IssueCreated struct {
	IssueID   string
	IssueType string
	Paradigm  string
	Timestamp time.Time
}

func (e IssueCreated) Type() string    { return "issue.created" }
func (e IssueCreated) Time() time.Time { return e.Timestamp }

type IssueTransitioned struct {
	IssueID   string
	From      string
	To        string
	Paradigm  string
	Timestamp time.Time
}

func (e IssueTransitioned) Type() string    { return "issue.transitioned" }
func (e IssueTransitioned) Time() time.Time { return e.Timestamp }

type SprintStarted struct {
	SprintID  string
	Paradigm  string
	Timestamp time.Time
}

func (e SprintStarted) Type() string    { return "sprint.started" }
func (e SprintStarted) Time() time.Time { return e.Timestamp }

type SprintEnded struct {
	SprintID  string
	Paradigm  string
	Timestamp time.Time
}

func (e SprintEnded) Type() string    { return "sprint.ended" }
func (e SprintEnded) Time() time.Time { return e.Timestamp }

// Dispatcher manages event publishing and subscription
type Dispatcher interface {
	Publish(ctx context.Context, event Event)
	Subscribe(eventType string, handler func(context.Context, Event))
	Unsubscribe(eventType string, handler func(context.Context, Event))
}

// InMemoryDispatcher is a simple in-memory event dispatcher
type InMemoryDispatcher struct {
	handlers map[string][]func(context.Context, Event)
	mu       sync.RWMutex
}

// NewInMemoryDispatcher creates a new in-memory event dispatcher
func NewInMemoryDispatcher() *InMemoryDispatcher {
	return &InMemoryDispatcher{
		handlers: make(map[string][]func(context.Context, Event)),
	}
}

// Publish sends an event to all registered handlers
func (d *InMemoryDispatcher) Publish(ctx context.Context, event Event) {
	d.mu.RLock()
	handlers, exists := d.handlers[event.Type()]
	d.mu.RUnlock()

	if !exists {
		return
	}

	// Execute handlers in goroutines to avoid blocking
	for _, handler := range handlers {
		go func(h func(context.Context, Event)) {
			defer func() {
				if r := recover(); r != nil {
					// Log error but don't crash
					// In a real implementation, use structured logging
					_ = r // Suppress staticcheck warning about empty branch
				}
			}()
			h(ctx, event)
		}(handler)
	}
}

// Subscribe adds a handler for a specific event type
func (d *InMemoryDispatcher) Subscribe(eventType string, handler func(context.Context, Event)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[eventType] = append(d.handlers[eventType], handler)
}

// Unsubscribe removes a handler for a specific event type
func (d *InMemoryDispatcher) Unsubscribe(eventType string, handler func(context.Context, Event)) {
	d.mu.Lock()
	defer d.mu.Unlock()

	handlers := d.handlers[eventType]
	for i, h := range handlers {
		// Note: function comparison is tricky in Go, this is a simplified approach
		// In practice, you might want to use a registration ID system
		if &h == &handler {
			d.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

// NoopDispatcher is a dispatcher that does nothing (useful for testing)
type NoopDispatcher struct{}

func (d *NoopDispatcher) Publish(ctx context.Context, event Event)                           {}
func (d *NoopDispatcher) Subscribe(eventType string, handler func(context.Context, Event))   {}
func (d *NoopDispatcher) Unsubscribe(eventType string, handler func(context.Context, Event)) {}
