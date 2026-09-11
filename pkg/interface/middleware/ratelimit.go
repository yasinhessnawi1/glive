package middleware

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/glive/interface/api"
	"github.com/gofiber/fiber/v2"
)

// bucket represents a token bucket for rate limiting
type bucket struct {
	tokens     int
	lastUpdate time.Time
	mu         sync.Mutex
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	requests    int           // max requests per window
	window      time.Duration // time window
	buckets     map[string]*bucket
	mu          sync.RWMutex
	cleanup     *time.Ticker
	stopCleanup chan bool
}

// NewRateLimiter creates a new rate limiter
// requests: maximum number of requests allowed
// window: time window for the limit (e.g., 1 * time.Minute)
func NewRateLimiter(requests int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests:    requests,
		window:      window,
		buckets:     make(map[string]*bucket),
		cleanup:     time.NewTicker(5 * time.Minute), // Clean up old buckets every 5 minutes
		stopCleanup: make(chan bool),
	}

	// Start cleanup goroutine
	go rl.cleanupBuckets()

	return rl
}

// Stop stops the rate limiter cleanup goroutine
func (rl *RateLimiter) Stop() {
	rl.cleanup.Stop()
	rl.stopCleanup <- true
}

// cleanupBuckets periodically removes old buckets
func (rl *RateLimiter) cleanupBuckets() {
	for {
		select {
		case <-rl.cleanup.C:
			rl.mu.Lock()
			now := time.Now()
			for key, b := range rl.buckets {
				b.mu.Lock()
				// Remove buckets that haven't been used in 2x the window
				if now.Sub(b.lastUpdate) > rl.window*2 {
					delete(rl.buckets, key)
				}
				b.mu.Unlock()
			}
			rl.mu.Unlock()
		case <-rl.stopCleanup:
			return
		}
	}
}

// getBucket gets or creates a bucket for a key
func (rl *RateLimiter) getBucket(key string) *bucket {
	rl.mu.RLock()
	b, exists := rl.buckets[key]
	rl.mu.RUnlock()

	if exists {
		return b
	}

	// Create new bucket
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if b, exists := rl.buckets[key]; exists {
		return b
	}

	b = &bucket{
		tokens:     rl.requests,
		lastUpdate: time.Now(),
	}
	rl.buckets[key] = b
	return b
}

// allow checks if a request should be allowed
// Returns true if allowed, false if rate limited
func (rl *RateLimiter) allow(key string) bool {
	b := rl.getBucket(key)

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastUpdate)

	// Refill tokens based on elapsed time
	if elapsed >= rl.window {
		// Full refill
		b.tokens = rl.requests
		b.lastUpdate = now
	} else {
		// Partial refill based on elapsed time
		tokensToAdd := int(float64(rl.requests) * elapsed.Seconds() / rl.window.Seconds())
		if tokensToAdd > 0 {
			b.tokens = min(b.tokens+tokensToAdd, rl.requests)
			b.lastUpdate = now
		}
	}

	// Check if we have tokens
	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// getKey extracts a key for rate limiting from Fiber context
// Uses IP address as the key
func (rl *RateLimiter) getKey(c *fiber.Ctx) string {
	ip := c.IP()

	// For localhost, use a single key to avoid issues with different representations
	if net.ParseIP(ip).IsLoopback() {
		return "localhost"
	}

	return ip
}

// Middleware returns a Fiber middleware function for rate limiting
func (rl *RateLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Ensure response context exists (in case rate limiter runs before context middleware)
		if c.Locals("response_context") == nil {
			ctx := api.NewResponseContext()
			api.SetResponseContext(c, ctx)
		}

		key := rl.getKey(c)

		if !rl.allow(key) {
			// Rate limited
			remaining := rl.getRemaining(key)
			resetTime := rl.getResetTime(key)
			retryAfter := int(time.Until(resetTime).Seconds())
			if retryAfter < 0 {
				retryAfter = 0
			}

			c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.requests))
			c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))
			c.Set("Retry-After", fmt.Sprintf("%d", retryAfter))

			// Return 429 Too Many Requests with standardized error format
			return api.TooManyRequests(c, "RATE_LIMIT_EXCEEDED", "Rate limit exceeded. Please try again later.", retryAfter)
		}

		// Set rate limit headers
		remaining := rl.getRemaining(key)
		resetTime := rl.getResetTime(key)

		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.requests))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

		return c.Next()
	}
}

// getRemaining gets remaining requests for a key
func (rl *RateLimiter) getRemaining(key string) int {
	b := rl.getBucket(key)
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastUpdate)

	if elapsed >= rl.window {
		return rl.requests
	}

	tokensToAdd := int(float64(rl.requests) * elapsed.Seconds() / rl.window.Seconds())
	currentTokens := min(b.tokens+tokensToAdd, rl.requests)
	return max(0, currentTokens)
}

// getResetTime gets the time when the rate limit resets
func (rl *RateLimiter) getResetTime(key string) time.Time {
	b := rl.getBucket(key)
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.lastUpdate.Add(rl.window)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
