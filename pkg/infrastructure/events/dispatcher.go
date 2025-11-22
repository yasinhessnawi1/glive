package events

import (
	"context"
	"fmt"
	"sync"

	"github.com/glive/domain/events"
)

// Dispatcher implements the EventDispatcher interface
type Dispatcher struct {
	handlers map[string][]events.EventHandler
	mu       sync.RWMutex
}

// NewDispatcher creates a new event dispatcher
func NewDispatcher() events.EventDispatcher {
	return &Dispatcher{
		handlers: make(map[string][]events.EventHandler),
	}
}

// Dispatch dispatches an event to all registered handlers
func (d *Dispatcher) Dispatch(ctx context.Context, event events.Event) error {
	d.mu.RLock()
	handlers := d.handlers[event.EventType()]
	d.mu.RUnlock()

	var errors []error
	for _, handler := range handlers {
		if err := handler.Handle(ctx, event); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors dispatching event %s: %v", event.EventType(), errors)
	}

	return nil
}

// Subscribe registers a handler for a specific event type
func (d *Dispatcher) Subscribe(eventType string, handler events.EventHandler) {
	if handler == nil {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.handlers[eventType] = append(d.handlers[eventType], handler)
}

// Unsubscribe removes a handler for a specific event type
func (d *Dispatcher) Unsubscribe(eventType string, handler events.EventHandler) {
	if handler == nil {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	handlers := d.handlers[eventType]
	for i, h := range handlers {
		if h == handler {
			d.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

