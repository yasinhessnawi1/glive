package api

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// APIResponse represents a standardized API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *APIMeta    `json:"meta,omitempty"`
}

// APIError represents an API error
type APIError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// APIMeta represents metadata about the API response
type APIMeta struct {
	RequestID string `json:"request_id"`
	Duration  string `json:"duration"`
}

// ResponseContext holds context for building responses
type ResponseContext struct {
	RequestID string
	StartTime time.Time
}

// NewResponseContext creates a new response context
func NewResponseContext() *ResponseContext {
	return &ResponseContext{
		RequestID: uuid.New().String(),
		StartTime: time.Now(),
	}
}

// SuccessResponse sends a successful API response
func SuccessResponse(c *fiber.Ctx, statusCode int, data interface{}) error {
	ctx := getOrCreateContext(c)
	duration := time.Since(ctx.StartTime)

	response := APIResponse{
		Success: true,
		Data:    data,
		Meta: &APIMeta{
			RequestID: ctx.RequestID,
			Duration:  formatDuration(duration),
		},
	}

	return c.Status(statusCode).JSON(response)
}

// ErrorResponse sends an error API response
func ErrorResponse(c *fiber.Ctx, statusCode int, code, message string, details map[string]interface{}) error {
	ctx := getOrCreateContext(c)
	duration := time.Since(ctx.StartTime)

	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: &APIMeta{
			RequestID: ctx.RequestID,
			Duration:  formatDuration(duration),
		},
	}

	return c.Status(statusCode).JSON(response)
}

// BadRequest sends a 400 Bad Request response
func BadRequest(c *fiber.Ctx, code, message string, details map[string]interface{}) error {
	return ErrorResponse(c, fiber.StatusBadRequest, code, message, details)
}

// NotFound sends a 404 Not Found response
func NotFound(c *fiber.Ctx, code, message string) error {
	return ErrorResponse(c, fiber.StatusNotFound, code, message, nil)
}

// InternalError sends a 500 Internal Server Error response
func InternalError(c *fiber.Ctx, code, message string) error {
	return ErrorResponse(c, fiber.StatusInternalServerError, code, message, nil)
}

// Unauthorized sends a 401 Unauthorized response
func Unauthorized(c *fiber.Ctx, code, message string) error {
	return ErrorResponse(c, fiber.StatusUnauthorized, code, message, nil)
}

// Forbidden sends a 403 Forbidden response
func Forbidden(c *fiber.Ctx, code, message string) error {
	return ErrorResponse(c, fiber.StatusForbidden, code, message, nil)
}

// TooManyRequests sends a 429 Too Many Requests response
func TooManyRequests(c *fiber.Ctx, code, message string, retryAfter int) error {
	c.Set("Retry-After", fmt.Sprintf("%d", retryAfter))
	details := map[string]interface{}{
		"retry_after": retryAfter,
	}
	return ErrorResponse(c, fiber.StatusTooManyRequests, code, message, details)
}

// getOrCreateContext retrieves or creates a response context from Fiber context
func getOrCreateContext(c *fiber.Ctx) *ResponseContext {
	// Try to get existing context
	if ctx, ok := c.Locals("response_context").(*ResponseContext); ok {
		return ctx
	}

	// Create new context
	ctx := NewResponseContext()
	c.Locals("response_context", ctx)
	return ctx
}

// formatDuration formats a duration as a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return d.String()
	}
	if d < time.Second {
		return d.Round(time.Millisecond).String()
	}
	return d.Round(time.Millisecond).String()
}

// SetResponseContext sets the response context in Fiber context (for middleware)
func SetResponseContext(c *fiber.Ctx, ctx *ResponseContext) {
	c.Locals("response_context", ctx)
}
