//go:build darwin

package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// MacOSSandbox implements Sandbox interface for macOS using sandbox-exec and resource limits
type MacOSSandbox struct {
	config      SandboxConfig
	rootPath    string
	initialized bool
}

// NewMacOSSandbox creates a new macOS sandbox instance
func NewMacOSSandbox() *MacOSSandbox {
	return &MacOSSandbox{}
}

// newPlatformSandbox creates a platform-specific sandbox (macOS implementation)
func newPlatformSandbox() (Sandbox, error) {
	return NewMacOSSandbox(), nil
}

// Initialize prepares the macOS sandbox environment
func (s *MacOSSandbox) Initialize(ctx context.Context, config SandboxConfig) error {
	if err := ValidateConfig(config); err != nil {
		return err
	}

	s.config = config
	s.rootPath = config.RootPath

	// Create sandbox root directory
	if err := os.MkdirAll(s.rootPath, 0755); err != nil {
		return fmt.Errorf("failed to create sandbox root: %w", err)
	}

	// Create work directory
	workDir := filepath.Join(s.rootPath, strings.TrimPrefix(config.WorkDir, "/"))
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}

	// Setup mount points
	for _, mount := range config.MountPoints {
		if err := ValidateMountPoint(s.rootPath, mount); err != nil {
			return err
		}

		dest := filepath.Join(s.rootPath, strings.TrimPrefix(mount.Destination, "/"))
		if err := os.MkdirAll(dest, 0755); err != nil {
			return fmt.Errorf("failed to create mount point: %w", err)
		}
	}

	s.initialized = true
	return nil
}

// Execute runs a command in the macOS sandbox
func (s *MacOSSandbox) Execute(ctx context.Context, cmd Command) (*Result, error) {
	if !s.initialized {
		return nil, fmt.Errorf("sandbox not initialized")
	}

	startTime := time.Now()

	// Parse resource limits
	memoryLimit, err := parseMemoryLimit(s.config.MemoryLimit)
	if err != nil {
		return nil, fmt.Errorf("invalid memory limit: %w", err)
	}

	// Build command
	execCmd := s.buildCommand(ctx, cmd)

	// Apply resource limits via ulimit
	if memoryLimit > 0 {
		// Set memory limit (RLIMIT_AS)
		var rlimit syscall.Rlimit
		rlimit.Cur = memoryLimit
		rlimit.Max = memoryLimit
		if err := syscall.Setrlimit(syscall.RLIMIT_AS, &rlimit); err != nil {
			// Log but don't fail - resource limits may not be available
			_ = err
		}
	}

	// Capture output
	var stdout, stderr strings.Builder
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	// Merge context timeout with config timeout
	timeoutCtx := ctx
	if s.config.Timeout > 0 {
		var cancel context.CancelFunc
		timeoutCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()
	}

	// Execute command
	err = execCmd.Run()
	duration := time.Since(startTime)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		}
	}

	// Get resource usage (approximate)
	cpuUsage, memoryUsed := s.getResourceUsage()

	return &Result{
		ExitCode:   exitCode,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		Success:    exitCode == 0 && err == nil,
		Error:      err,
		Duration:   duration,
		CPUUsage:   cpuUsage,
		MemoryUsed: memoryUsed,
	}, nil
}

// Cleanup removes sandbox artifacts
func (s *MacOSSandbox) Cleanup() error {
	// Remove sandbox directory
	if s.rootPath != "" {
		return os.RemoveAll(s.rootPath)
	}

	return nil
}

// buildCommand builds the command with sandbox-exec if available
func (s *MacOSSandbox) buildCommand(ctx context.Context, cmd Command) *exec.Cmd {
	// Set working directory
	workDir := filepath.Join(s.rootPath, strings.TrimPrefix(cmd.WorkingDir, "/"))
	if workDir == "" || cmd.WorkingDir == "" {
		workDir = filepath.Join(s.rootPath, strings.TrimPrefix(s.config.WorkDir, "/"))
	}

	var execCmd *exec.Cmd

	// Try to use sandbox-exec if available
	if s.hasSandboxExec() {
		// Generate seatbelt profile
		profile := s.generateSeatbeltProfile()

		// Use sandbox-exec with generated profile
		args := []string{"-f", "-", cmd.Command}
		args = append(args, cmd.Args...)

		execCmd = exec.CommandContext(ctx, "sandbox-exec", args...)
		execCmd.Stdin = strings.NewReader(profile)
	} else {
		// Fallback to regular command execution with resource limits
		execCmd = exec.CommandContext(ctx, cmd.Command, cmd.Args...)
	}

	execCmd.Dir = workDir

	// Set environment
	env := os.Environ()
	for k, v := range cmd.Environment {
		env = append(env, k+"="+v)
	}
	for k, v := range s.config.Environment {
		env = append(env, k+"="+v)
	}
	execCmd.Env = env

	return execCmd
}

// hasSandboxExec checks if sandbox-exec is available
func (s *MacOSSandbox) hasSandboxExec() bool {
	_, err := exec.LookPath("sandbox-exec")
	return err == nil
}

// generateSeatbeltProfile generates a seatbelt profile for sandbox-exec
func (s *MacOSSandbox) generateSeatbeltProfile() string {
	var profile strings.Builder

	profile.WriteString("(version 1)\n")
	profile.WriteString("(deny default)\n")

	// Allow file read/write in sandbox root
	profile.WriteString(fmt.Sprintf("(allow file-read* file-write* (subpath \"%s\"))\n", s.rootPath))

	// Allow network if enabled
	if s.config.AllowNetwork {
		profile.WriteString("(allow network-outbound)\n")
		if s.config.AllowDNS {
			profile.WriteString("(allow dns-lookup)\n")
		}
	}

	// Allow process execution
	profile.WriteString("(allow process-exec)\n")

	// Allow system calls needed for basic operations
	profile.WriteString("(allow sysctl-read)\n")
	profile.WriteString("(allow mach-lookup)\n")

	return profile.String()
}

// getResourceUsage gets approximate resource usage
func (s *MacOSSandbox) getResourceUsage() (float64, int64) {
	// On macOS, getting precise resource usage requires more complex APIs
	// For now, return approximate values
	var cpuUsage float64
	var memoryUsed int64

	// Try to get process stats using ps or similar
	// This is a simplified implementation
	// In production, you might want to use libproc or similar

	return cpuUsage, memoryUsed
}

// parseMemoryLimit parses memory limit string ("512MB", "2GB", "1024")
func parseMemoryLimit(limitStr string) (int64, error) {
	limitStr = strings.TrimSpace(strings.ToUpper(limitStr))
	if limitStr == "" {
		return 0, nil
	}

	var multiplier int64 = 1
	if strings.HasSuffix(limitStr, "KB") {
		multiplier = 1024
		limitStr = strings.TrimSuffix(limitStr, "KB")
	} else if strings.HasSuffix(limitStr, "MB") {
		multiplier = 1024 * 1024
		limitStr = strings.TrimSuffix(limitStr, "MB")
	} else if strings.HasSuffix(limitStr, "GB") {
		multiplier = 1024 * 1024 * 1024
		limitStr = strings.TrimSuffix(limitStr, "GB")
	} else if strings.HasSuffix(limitStr, "TB") {
		multiplier = 1024 * 1024 * 1024 * 1024
		limitStr = strings.TrimSuffix(limitStr, "TB")
	}

	value, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory limit format: %w", err)
	}

	return value * multiplier, nil
}


