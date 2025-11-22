package tracing

import (
	"github.com/gofiber/fiber/v2"
)

// FiberMiddleware creates Fiber middleware for tracing requests
func FiberMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Create span for the request
		ctx, span := StartSpan(c.Context(), "http.request")
		defer func() {
			// Set status based on response status code
			status := SpanStatusOK
			if c.Response().StatusCode() >= 400 {
				status = SpanStatusError
			}
			EndSpan(span, status)
		}()

		// Add trace attributes
		span.SetAttribute("http.method", c.Method())
		span.SetAttribute("http.path", c.Path())
		span.SetAttribute("http.route", c.Route().Path)
		span.SetAttribute("http.remote_addr", c.IP())
		span.SetAttribute("http.user_agent", c.Get("User-Agent"))

		// Update context
		c.SetUserContext(ctx)

		// Add trace headers to response
		traceID := string(GetTraceID(ctx))
		spanID := string(GetSpanID(ctx))
		if traceID != "" {
			c.Set("X-Trace-ID", traceID)
		}
		if spanID != "" {
			c.Set("X-Span-ID", spanID)
		}

		// Call next handler
		err := c.Next()

		// Set final status code attribute
		span.SetAttribute("http.status_code", c.Response().StatusCode())

		return err
	}
}
