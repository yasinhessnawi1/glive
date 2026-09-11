package events

import "context"

// EventHandler handles domain events
type EventHandler interface {
	Handle(ctx context.Context, event Event) error
}

// EventHandlerFunc is a function that implements EventHandler
type EventHandlerFunc func(ctx context.Context, event Event) error

func (f EventHandlerFunc) Handle(ctx context.Context, event Event) error {
	return f(ctx, event)
}

// EventDispatcher dispatches domain events to registered handlers
type EventDispatcher interface {
	// Dispatch dispatches an event to all registered handlers
	Dispatch(ctx context.Context, event Event) error

	// Subscribe registers a handler for a specific event type
	Subscribe(eventType string, handler EventHandler)

	// Unsubscribe removes a handler for a specific event type
	Unsubscribe(eventType string, handler EventHandler)
}
