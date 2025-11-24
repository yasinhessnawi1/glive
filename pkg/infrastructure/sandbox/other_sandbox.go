//go:build !linux && !windows && !darwin

package sandbox

import (
	"context"
	"errors"
)

// UnsupportedSandbox is a stub for platforms without sandbox support
type UnsupportedSandbox struct{}

// newPlatformSandbox returns an error for unsupported platforms
func newPlatformSandbox() (Sandbox, error) {
	return &UnsupportedSandbox{}, nil
}

// Initialize returns an error as sandboxing is not supported
func (s *UnsupportedSandbox) Initialize(ctx context.Context, config SandboxConfig) error {
	return errors.New("sandbox not supported on this platform")
}

// Execute returns an error as sandboxing is not supported
func (s *UnsupportedSandbox) Execute(ctx context.Context, cmd Command) (*Result, error) {
	return nil, errors.New("sandbox not supported on this platform")
}

// Cleanup is a no-op for unsupported platforms
func (s *UnsupportedSandbox) Cleanup() error {
	return nil
}
