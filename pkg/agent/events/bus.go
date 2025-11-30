package events

import (
	"sync"
	"time"
)

const (
	// MaxBufferSize is the maximum number of events to buffer per project
	MaxBufferSize = 500
	// BufferRetention is how long to keep buffered events
	BufferRetention = 30 * time.Minute
)

// BufferedEvent wraps an event with a timestamp for cleanup
type BufferedEvent struct {
	Event     interface{}
	Timestamp time.Time
}

// Bus manages event subscriptions and publishing with buffering
type Bus struct {
	subscribers map[string][]chan<- interface{}
	// eventBuffer stores recent events per project for replay on reconnect
	eventBuffer map[string][]BufferedEvent
	mu          sync.RWMutex
}

// NewBus creates a new event bus
func NewBus() *Bus {
	bus := &Bus{
		subscribers: make(map[string][]chan<- interface{}),
		eventBuffer: make(map[string][]BufferedEvent),
	}
	// Start cleanup goroutine
	go bus.cleanupLoop()
	return bus
}

// cleanupLoop periodically removes old buffered events
func (b *Bus) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		b.cleanupOldEvents()
	}
}

// cleanupOldEvents removes events older than BufferRetention
func (b *Bus) cleanupOldEvents() {
	b.mu.Lock()
	defer b.mu.Unlock()

	cutoff := time.Now().Add(-BufferRetention)
	for projectID, events := range b.eventBuffer {
		// Find first event that's not too old
		firstValid := 0
		for i, e := range events {
			if e.Timestamp.After(cutoff) {
				firstValid = i
				break
			}
			firstValid = i + 1
		}
		if firstValid >= len(events) {
			// All events are old, remove project buffer
			delete(b.eventBuffer, projectID)
		} else if firstValid > 0 {
			// Remove old events
			b.eventBuffer[projectID] = events[firstValid:]
		}
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

// Publish sends an event to all subscribers of a project and buffers it
func (b *Bus) Publish(projectID string, event interface{}) {
	b.publishInternal(projectID, event, true)
}

// PublishWithoutBuffer sends an event to all subscribers but does NOT buffer it (for log lines)
func (b *Bus) PublishWithoutBuffer(projectID string, event interface{}) {
	b.publishInternal(projectID, event, false)
}

// publishInternal is the internal publish method with optional buffering
func (b *Bus) publishInternal(projectID string, event interface{}, shouldBuffer bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Buffer the event for replay (only if shouldBuffer is true)
	if shouldBuffer {
		bufferedEvent := BufferedEvent{
			Event:     event,
			Timestamp: time.Now(),
		}

		if _, exists := b.eventBuffer[projectID]; !exists {
			b.eventBuffer[projectID] = make([]BufferedEvent, 0, MaxBufferSize)
		}

		buffer := b.eventBuffer[projectID]
		// Maintain max buffer size
		if len(buffer) >= MaxBufferSize {
			// Remove oldest events (keep last MaxBufferSize-1)
			buffer = buffer[len(buffer)-MaxBufferSize+1:]
		}
		b.eventBuffer[projectID] = append(buffer, bufferedEvent)
	}

	// Send to subscribers
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

// GetBufferedEvents returns all buffered events for a project (for replay on reconnect)
func (b *Bus) GetBufferedEvents(projectID string) []interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	buffer, exists := b.eventBuffer[projectID]
	if !exists {
		return nil
	}

	events := make([]interface{}, len(buffer))
	for i, be := range buffer {
		events[i] = be.Event
	}
	return events
}

// ClearBuffer clears buffered events for a project (call when project completes or is deleted)
func (b *Bus) ClearBuffer(projectID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.eventBuffer, projectID)
}
