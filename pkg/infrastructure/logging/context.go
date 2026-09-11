package logging

import "context"

// getTraceIDFromContext extracts trace ID from context (avoiding circular dependency with tracing package)
func getTraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	// Check for trace_id key (set by tracing package via context.WithValue)
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		return traceID
	}
	return ""
}

// getSpanIDFromContext extracts span ID from context (avoiding circular dependency with tracing package)
func getSpanIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	// Check for span_id key (set by tracing package via context.WithValue)
	if spanID, ok := ctx.Value("span_id").(string); ok {
		return spanID
	}
	return ""
}
