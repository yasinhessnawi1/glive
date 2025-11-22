package handlers

import (
	"github.com/gofiber/fiber/v2"
)

var globalMetrics Metrics

// Metrics is the interface for accessing metrics
type Metrics interface {
	Export() string
}

// SetMetrics sets the global metrics instance
func SetMetrics(m Metrics) {
	globalMetrics = m
}

// MetricsHandler handles the /metrics endpoint for Prometheus scraping
func MetricsHandler(c *fiber.Ctx) error {
	if globalMetrics == nil {
		// Return empty metrics if not initialized
		c.Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		return c.SendString("# Metrics not available\n")
	}

	c.Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	return c.SendString(globalMetrics.Export())
}

