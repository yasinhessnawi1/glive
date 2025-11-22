package events

import (
	"sync"
)

// Bus manages event subscriptions and publishing
type Bus struct {
	subscribers map[string][]chan<- interface{}
	mu          sync.RWMutex
}

// NewBus creates a new event bus
func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[string][]chan<- interface{}),
	}
}

// Subscribe adds a channel to receive events for a specific project
func (b *Bus) Subscribe(projectID string, ch chan<- interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.subscribers[projectID]; !exists {
		b.subscribers[projectID] = []chan<- interface{}{}
	}
	b.subscribers[projectID] = append(b.subscribers[projectID], ch)
}

// Unsubscribe removes a channel from subscriptions
func (b *Bus) Unsubscribe(projectID string, ch chan<- interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if subs, exists := b.subscribers[projectID]; exists {
		for i, sub := range subs {
			if sub == ch {
				// Remove element at index i
				b.subscribers[projectID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		// Clean up if empty
		if len(b.subscribers[projectID]) == 0 {
			delete(b.subscribers, projectID)
		}
	}
}

// Publish sends an event to all subscribers of a project
func (b *Bus) Publish(projectID string, event interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if subs, exists := b.subscribers[projectID]; exists {
		for _, ch := range subs {
			// Non-blocking send to avoid blocking the publisher
			select {
			case ch <- event:
			default:
				// Channel full or blocked, skip (or log warning)
			}
		}
	}
}
