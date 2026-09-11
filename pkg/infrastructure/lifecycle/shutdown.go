package lifecycle

import (
	"context"
	"sync"
	"time"
)

// ShutdownManager manages graceful shutdown of goroutines and resources
// It ensures all registered goroutines complete before shutdown completes
type ShutdownManager struct {
	wg      sync.WaitGroup
	done    chan struct{}
	timeout time.Duration
	mu      sync.Mutex
	closed  bool
}

// NewShutdownManager creates a new shutdown manager with the specified timeout
func NewShutdownManager(timeout time.Duration) *ShutdownManager {
	return &ShutdownManager{
		done:    make(chan struct{}),
		timeout: timeout,
	}
}

// Register adds a goroutine to the wait group
// Call this before starting a goroutine that should be waited on during shutdown
func (m *ShutdownManager) Register() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.wg.Add(1)
	}
}

// Done marks a registered goroutine as complete
// Call this when a registered goroutine finishes
func (m *ShutdownManager) Done() {
	m.wg.Done()
}

// Shutdown gracefully shuts down all registered goroutines
// It closes the done channel and waits for all registered goroutines to complete
// Returns an error if the context times out before all goroutines complete
func (m *ShutdownManager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	close(m.done)
	m.mu.Unlock()

	// Create a channel to signal when WaitGroup is done
	c := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(c)
	}()

	// Wait for either completion or timeout
	select {
	case <-c:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ShutdownWithTimeout gracefully shuts down with the manager's configured timeout
func (m *ShutdownManager) ShutdownWithTimeout() error {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()
	return m.Shutdown(ctx)
}

// DoneChannel returns the done channel
// Goroutines can listen to this channel to know when shutdown has been initiated
func (m *ShutdownManager) DoneChannel() <-chan struct{} {
	return m.done
}

// IsShuttingDown returns true if shutdown has been initiated
func (m *ShutdownManager) IsShuttingDown() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}
