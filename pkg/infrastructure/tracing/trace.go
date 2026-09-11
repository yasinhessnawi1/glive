package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// TraceID represents a unique trace identifier
type TraceID string

// SpanID represents a unique span identifier
type SpanID string

// SpanStatus represents the status of a span
type SpanStatus int

const (
	SpanStatusOK SpanStatus = iota
	SpanStatusError
	SpanStatusUnset
)

// String returns the string representation of the span status
func (s SpanStatus) String() string {
	switch s {
	case SpanStatusOK:
		return "OK"
	case SpanStatusError:
		return "ERROR"
	default:
		return "UNSET"
	}
}

// Span represents a single operation in a trace
type Span struct {
	TraceID    TraceID
	SpanID     SpanID
	ParentID   SpanID
	Name       string
	StartTime  time.Time
	EndTime    time.Time
	Status     SpanStatus
	Attributes map[string]interface{}
	mu         sync.Mutex
}

// NewTraceID generates a new trace ID
func NewTraceID() TraceID {
	return TraceID(generateID())
}

// NewSpanID generates a new span ID
func NewSpanID() SpanID {
	return SpanID(generateID())
}

// generateID generates a random 16-byte hex string
func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// StartSpan starts a new span in the given context
func StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	span := &Span{
		SpanID:     NewSpanID(),
		Name:       name,
		StartTime:  time.Now(),
		Status:     SpanStatusUnset,
		Attributes: make(map[string]interface{}),
	}

	// Get trace ID and parent span ID from context
	if parentTraceID := GetTraceID(ctx); parentTraceID != "" {
		span.TraceID = parentTraceID
	} else {
		span.TraceID = NewTraceID()
	}

	if parentSpanID := GetSpanID(ctx); parentSpanID != "" {
		span.ParentID = parentSpanID
	}

	// Create new context with span information
	ctx = WithTraceID(ctx, span.TraceID)
	ctx = WithSpanID(ctx, span.SpanID)
	ctx = context.WithValue(ctx, spanContextKey{}, span)

	return ctx, span
}

// EndSpan ends a span and sets its status
func EndSpan(span *Span, status SpanStatus) {
	if span == nil {
		return
	}

	span.mu.Lock()
	defer span.mu.Unlock()

	span.EndTime = time.Now()
	span.Status = status
}

// SetAttribute sets an attribute on the span
func (s *Span) SetAttribute(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Attributes == nil {
		s.Attributes = make(map[string]interface{})
	}
	s.Attributes[key] = value
}

// SetStatus sets the status of the span
func (s *Span) SetStatus(status SpanStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Status = status
}

// Duration returns the duration of the span
func (s *Span) Duration() time.Duration {
	if s.EndTime.IsZero() {
		return time.Since(s.StartTime)
	}
	return s.EndTime.Sub(s.StartTime)
}

// IsRecording returns true if the span is still recording
func (s *Span) IsRecording() bool {
	return s.EndTime.IsZero()
}

// String returns a string representation of the span
func (s *Span) String() string {
	return fmt.Sprintf("Span{name=%s, traceID=%s, spanID=%s, duration=%v, status=%s}",
		s.Name, s.TraceID, s.SpanID, s.Duration(), s.Status)
}
