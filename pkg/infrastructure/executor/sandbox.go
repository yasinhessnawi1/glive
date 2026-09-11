package executor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glive/infrastructure/sandbox"
	"github.com/glive/infrastructure/security"
)

// SandboxConfig defines sandbox configuration
type SandboxConfig struct {
	RootPath                  string
	EnableNetworkIsolation    bool
	EnableFileSystemIsolation bool
	AllowedPaths              []string
	DeniedPaths               []string
	ResourceLimits            ResourceLimits
	Timeout                   time.Duration
}

// ResourceLimits defines resource limits for sandboxed execution
type ResourceLimits struct {
	MaxCPUPercent  int
	MaxMemoryMB    int
	MaxDiskWriteMB int
	MaxProcesses   int
}

// DefaultSandboxConfig returns a default sandbox configuration
func DefaultSandboxConfig() SandboxConfig {
	return SandboxConfig{
		EnableNetworkIsolation:    false, // Disabled by default
		EnableFileSystemIsolation: true,
		ResourceLimits: ResourceLimits{
			MaxCPUPercent:  50,
			MaxMemoryMB:    1024, // 1GB
			MaxDiskWriteMB: 100,
			MaxProcesses:   10,
		},
		Timeout: 10 * time.Minute,
	}
}

// SandboxedExecutor provides sandboxed command execution
type SandboxedExecutor struct {
	config      SandboxConfig
	validator   *security.CommandValidator
	sandbox     sandbox.Sandbox
	initialized bool
}

// NewSandboxedExecutor creates a new sandboxed executor
func NewSandboxedExecutor(config SandboxConfig) *SandboxedExecutor {
	return &SandboxedExecutor{
		config:      config,
		validator:   security.NewCommandValidator(),
		initialized: false,
	}
}

// Execute executes a command in a sandbox
// All long-running operations MUST accept context for cancellation
func (e *SandboxedExecutor) Execute(ctx context.Context, cmd *Command, outputHandler OutputHandler) error {
	// Check for context cancellation before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Validate command first
	if err := e.validator.Validate(cmd.Command); err != nil {
		return fmt.Errorf("command validation failed: %w", err)
	}

	// Initialize sandbox if not already done
	if !e.initialized {
		if err := e.initializeSandbox(ctx); err != nil {
			return fmt.Errorf("failed to initialize sandbox: %w", err)
		}
	}

	// Parse command
	parts := strings.Fields(cmd.Command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	// Convert to sandbox command format
	sandboxCmd := sandbox.Command{
		Command:     parts[0],
		Args:        parts[1:],
		WorkingDir:  cmd.WorkingDir,
		Environment: make(map[string]string),
	}

	// Execute in sandbox
	result, err := e.sandbox.Execute(ctx, sandboxCmd)
	if err != nil {
		return fmt.Errorf("sandbox execution failed: %w", err)
	}

	// Stream output to handler
	if outputHandler != nil {
		for _, line := range strings.Split(result.Stdout, "\n") {
			if line != "" {
				outputHandler(line)
			}
		}
		for _, line := range strings.Split(result.Stderr, "\n") {
			if line != "" {
				outputHandler("[ERROR] " + line)
			}
		}
	}

	// Update command status
	if result.Success {
		cmd.Status = CommandCompleted
		cmd.ExitCode = result.ExitCode
		cmd.Output = result.Stdout
	} else {
		cmd.Status = CommandFailed
		cmd.ExitCode = result.ExitCode
		cmd.Error = result.Stderr
		if result.Error != nil {
			return result.Error
		}
		return fmt.Errorf("command failed with exit code %d", result.ExitCode)
	}

	return nil
}

// initializeSandbox initializes the platform-specific sandbox
func (e *SandboxedExecutor) initializeSandbox(ctx context.Context) error {
	// Create sandbox instance
	sb, err := sandbox.NewSandbox()
	if err != nil {
		return fmt.Errorf("failed to create sandbox: %w", err)
	}

	// Convert executor config to sandbox config
	sandboxConfig := e.convertToSandboxConfig()

	// Initialize sandbox
	if err := sb.Initialize(ctx, sandboxConfig); err != nil {
		return fmt.Errorf("failed to initialize sandbox: %w", err)
	}

	e.sandbox = sb
	e.initialized = true

	return nil
}

// convertToSandboxConfig converts executor SandboxConfig to sandbox.SandboxConfig
func (e *SandboxedExecutor) convertToSandboxConfig() sandbox.SandboxConfig {
	// Determine sandbox root
	rootPath := e.config.RootPath
	if rootPath == "" {
		tmpDir := os.TempDir()
		if tmpDir == "" {
			tmpDir = "/tmp"
		}
		rootPath = filepath.Join(tmpDir, fmt.Sprintf("glive-sandbox-%d", os.Getpid()))
	}

	// Convert CPU limit
	cpuLimit := fmt.Sprintf("%d%%", e.config.ResourceLimits.MaxCPUPercent)

	// Convert memory limit
	memoryLimit := fmt.Sprintf("%dMB", e.config.ResourceLimits.MaxMemoryMB)

	return sandbox.SandboxConfig{
		RootPath:     rootPath,
		WorkDir:      "/work",
		AllowNetwork: !e.config.EnableNetworkIsolation,
		AllowDNS:     !e.config.EnableNetworkIsolation,
		CPULimit:     cpuLimit,
		MemoryLimit:  memoryLimit,
		Timeout:      e.config.Timeout,
		Environment:  make(map[string]string),
		MountPoints:  []sandbox.MountPoint{},
	}
}

// ExecuteMultiple executes multiple commands in a sandbox
func (e *SandboxedExecutor) ExecuteMultiple(ctx context.Context, commands []*Command, outputHandler OutputHandler) error {
	// Initialize sandbox if not already done
	if !e.initialized {
		if err := e.initializeSandbox(ctx); err != nil {
			return fmt.Errorf("failed to initialize sandbox: %w", err)
		}
	}

	for _, cmd := range commands {
		// Check for context cancellation before each command
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := e.Execute(ctx, cmd, outputHandler); err != nil {
			return err
		}
	}

	// Cleanup sandbox after all commands
	if e.sandbox != nil {
		_ = e.sandbox.Cleanup()
		e.initialized = false
	}

	return nil
}

// CheckSandboxAvailable checks if sandboxing is available on this platform
func CheckSandboxAvailable() bool {
	return sandbox.CheckSandboxAvailable()
}
