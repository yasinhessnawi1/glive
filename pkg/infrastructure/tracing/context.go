package tracing

import (
	"context"
)

type traceIDKey struct{}
type spanIDKey struct{}
type spanContextKey struct{}

// WithTraceID adds a trace ID to the context
func WithTraceID(ctx context.Context, traceID TraceID) context.Context {
	// Also set as string for logging package compatibility
	ctx = context.WithValue(ctx, traceIDKey{}, traceID)
	ctx = context.WithValue(ctx, "trace_id", string(traceID))
	return ctx
}

// GetTraceID retrieves the trace ID from the context
func GetTraceID(ctx context.Context) TraceID {
	if ctx == nil {
		return ""
	}
	if traceID, ok := ctx.Value(traceIDKey{}).(TraceID); ok {
		return traceID
	}
	// Fallback to string value for compatibility
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		return TraceID(traceID)
	}
	return ""
}

// WithSpanID adds a span ID to the context
func WithSpanID(ctx context.Context, spanID SpanID) context.Context {
	// Also set as string for logging package compatibility
	ctx = context.WithValue(ctx, spanIDKey{}, spanID)
	ctx = context.WithValue(ctx, "span_id", string(spanID))
	return ctx
}

// GetSpanID retrieves the span ID from the context
func GetSpanID(ctx context.Context) SpanID {
	if ctx == nil {
		return ""
	}
	if spanID, ok := ctx.Value(spanIDKey{}).(SpanID); ok {
		return spanID
	}
	// Fallback to string value for compatibility
	if spanID, ok := ctx.Value("span_id").(string); ok {
		return SpanID(spanID)
	}
	return ""
}

// GetSpan retrieves the current span from the context
func GetSpan(ctx context.Context) *Span {
	if ctx == nil {
		return nil
	}
	if span, ok := ctx.Value(spanContextKey{}).(*Span); ok {
		return span
	}
	return nil
}
