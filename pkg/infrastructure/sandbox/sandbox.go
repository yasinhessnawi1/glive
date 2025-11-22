package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

// Sandbox provides a unified interface for platform-specific sandbox implementations
type Sandbox interface {
	// Initialize prepares the sandbox environment
	Initialize(ctx context.Context, config SandboxConfig) error

	// Execute runs a command in the sandbox
	Execute(ctx context.Context, cmd Command) (*Result, error)

	// Cleanup removes sandbox artifacts
	Cleanup() error
}

// SandboxConfig defines sandbox configuration
type SandboxConfig struct {
	RootPath     string            // Sandbox root directory
	WorkDir      string            // Working directory inside sandbox
	AllowNetwork bool              // Enable network access
	AllowDNS     bool              // Enable DNS resolution
	CPULimit     string            // "1" = 1 core, "50%" = half core
	MemoryLimit  string            // "512MB", "2GB"
	Timeout      time.Duration     // Max execution time
	Environment  map[string]string // Environment variables
	MountPoints  []MountPoint      // Filesystem mount points
}

// MountPoint defines a filesystem mount point
type MountPoint struct {
	Source      string // Source path (host)
	Destination string // Destination path (sandbox)
	ReadOnly    bool   // Mount as read-only
}

// Command represents a command to execute in the sandbox
type Command struct {
	Command     string            // Command to execute
	Args        []string          // Command arguments
	WorkingDir  string            // Working directory (relative to sandbox root)
	Environment map[string]string // Additional environment variables
}

// Result represents the result of command execution
type Result struct {
	ExitCode   int    // Process exit code
	Stdout     string // Standard output
	Stderr     string // Standard error
	Success    bool   // Whether command succeeded
	Error      error  // Execution error if any
	Duration   time.Duration
	CPUUsage   float64 // CPU usage percentage
	MemoryUsed int64   // Memory used in bytes
}

// DefaultSandboxConfig returns a default sandbox configuration
func DefaultSandboxConfig() SandboxConfig {
	tmpDir := os.TempDir()
	if tmpDir == "" {
		tmpDir = "/tmp"
	}

	// Generate unique sandbox root directory
	sandboxRoot := filepath.Join(tmpDir, "glive-sandbox")

	return SandboxConfig{
		RootPath:     sandboxRoot,
		WorkDir:      "/work",
		AllowNetwork: true,
		AllowDNS:     true,
		CPULimit:     "1",
		MemoryLimit:  "1GB",
		Timeout:      30 * time.Minute,
		Environment:  make(map[string]string),
		MountPoints:  []MountPoint{},
	}
}

// ValidateConfig validates sandbox configuration
func ValidateConfig(config SandboxConfig) error {
	if config.RootPath == "" {
		return ErrInvalidConfig{Field: "RootPath", Reason: "cannot be empty"}
	}

	// Validate RootPath is absolute
	if !filepath.IsAbs(config.RootPath) {
		return ErrInvalidConfig{Field: "RootPath", Reason: "must be absolute path"}
	}

	// Validate timeout
	if config.Timeout <= 0 {
		return ErrInvalidConfig{Field: "Timeout", Reason: "must be positive"}
	}

	return nil
}

// ErrInvalidConfig represents a configuration error
type ErrInvalidConfig struct {
	Field  string
	Reason string
}

func (e ErrInvalidConfig) Error() string {
	return "invalid sandbox config: " + e.Field + ": " + e.Reason
}


