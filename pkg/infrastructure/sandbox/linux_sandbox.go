//go:build linux

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

// LinuxSandbox implements Sandbox interface for Linux using namespaces and cgroups
type LinuxSandbox struct {
	config      SandboxConfig
	rootPath    string
	cgroupPath  string
	initialized bool
}

// NewLinuxSandbox creates a new Linux sandbox instance
func NewLinuxSandbox() *LinuxSandbox {
	return &LinuxSandbox{}
}

// newPlatformSandbox creates a platform-specific sandbox (Linux implementation)
func newPlatformSandbox() (Sandbox, error) {
	return NewLinuxSandbox(), nil
}

// Initialize prepares the Linux sandbox environment
func (s *LinuxSandbox) Initialize(ctx context.Context, config SandboxConfig) error {
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

		// Bind mount will be done in Execute when we have namespaces
	}

	s.initialized = true
	return nil
}

// Execute runs a command in the Linux sandbox
func (s *LinuxSandbox) Execute(ctx context.Context, cmd Command) (*Result, error) {
	if !s.initialized {
		return nil, fmt.Errorf("sandbox not initialized")
	}

	startTime := time.Now()

	// Create cgroup for resource limits
	cgroupPath, err := s.createCgroup()
	if err != nil {
		return nil, fmt.Errorf("failed to create cgroup: %w", err)
	}
	s.cgroupPath = cgroupPath
	defer s.cleanupCgroup()

	// Parse resource limits
	cpuLimit, err := parseCPULimit(s.config.CPULimit)
	if err != nil {
		return nil, fmt.Errorf("invalid CPU limit: %w", err)
	}

	memoryLimit, err := parseMemoryLimit(s.config.MemoryLimit)
	if err != nil {
		return nil, fmt.Errorf("invalid memory limit: %w", err)
	}

	// Apply cgroup limits
	if err := s.applyCgroupLimits(cgroupPath, cpuLimit, memoryLimit); err != nil {
		return nil, fmt.Errorf("failed to apply cgroup limits: %w", err)
	}

	// Build command with unshare for namespaces
	execCmd := s.buildCommand(ctx, cmd, cgroupPath)

	// Capture output
	var stdout, stderr strings.Builder
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	// Merge context timeout with config timeout
	if s.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.config.Timeout)
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

	// Get resource usage from cgroup
	cpuUsage, memoryUsed := s.getResourceUsage(cgroupPath)

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
func (s *LinuxSandbox) Cleanup() error {
	if s.cgroupPath != "" {
		if err := s.cleanupCgroup(); err != nil {
			return err
		}
	}

	// Remove sandbox directory
	if s.rootPath != "" {
		return os.RemoveAll(s.rootPath)
	}

	return nil
}

// buildCommand builds the command with namespace isolation
func (s *LinuxSandbox) buildCommand(ctx context.Context, cmd Command, cgroupPath string) *exec.Cmd {
	// Set working directory
	workDir := filepath.Join(s.rootPath, strings.TrimPrefix(cmd.WorkingDir, "/"))
	if workDir == "" || cmd.WorkingDir == "" {
		workDir = filepath.Join(s.rootPath, strings.TrimPrefix(s.config.WorkDir, "/"))
	}

	// Create command
	execCmd := exec.CommandContext(ctx, cmd.Command, cmd.Args...)
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

	// Set syscall attributes for namespace isolation
	cloneFlags := syscall.CLONE_NEWNS | syscall.CLONE_NEWPID
	if !s.config.AllowNetwork {
		cloneFlags |= syscall.CLONE_NEWNET
	}

	execCmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: uintptr(cloneFlags),
		// Set cgroup path in child process
		UseCgroupFD: true,
	}

	return execCmd
}

// createCgroup creates a cgroup for this sandbox
func (s *LinuxSandbox) createCgroup() (string, error) {
	// Try cgroup v2 first (most modern systems)
	cgroupV2Path := "/sys/fs/cgroup"
	if _, err := os.Stat(cgroupV2Path); err == nil {
		// Create unique cgroup name
		cgroupName := fmt.Sprintf("glive-%d", os.Getpid())
		cgroupPath := filepath.Join(cgroupV2Path, cgroupName)

		if err := os.MkdirAll(cgroupPath, 0755); err != nil {
			return "", fmt.Errorf("failed to create cgroup: %w", err)
		}

		return cgroupPath, nil
	}

	// Fallback: try to use systemd user cgroup
	// For now, return error if cgroup v2 not available
	return "", fmt.Errorf("cgroup v2 not available")
}

// applyCgroupLimits applies CPU and memory limits to cgroup
func (s *LinuxSandbox) applyCgroupLimits(cgroupPath string, cpuLimit float64, memoryLimit int64) error {
	// Set CPU limit (cpu.max format: "1 100000" for 1 CPU, or "50000 100000" for 50%)
	if cpuLimit > 0 {
		cpuMax := fmt.Sprintf("%d 100000", int(cpuLimit*100000))
		cpuMaxPath := filepath.Join(cgroupPath, "cpu.max")
		if err := os.WriteFile(cpuMaxPath, []byte(cpuMax), 0644); err != nil {
			return fmt.Errorf("failed to set CPU limit: %w", err)
		}
	}

	// Set memory limit
	if memoryLimit > 0 {
		memoryMaxPath := filepath.Join(cgroupPath, "memory.max")
		if err := os.WriteFile(memoryMaxPath, []byte(strconv.FormatInt(memoryLimit, 10)), 0644); err != nil {
			return fmt.Errorf("failed to set memory limit: %w", err)
		}
	}

	return nil
}

// getResourceUsage reads resource usage from cgroup
func (s *LinuxSandbox) getResourceUsage(cgroupPath string) (float64, int64) {
	var cpuUsage float64
	var memoryUsed int64

	// Read CPU usage (cpu.stat)
	cpuStatPath := filepath.Join(cgroupPath, "cpu.stat")
	if data, err := os.ReadFile(cpuStatPath); err == nil {
		// Parse cpu.stat for usage
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "usage_usec") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					if usage, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
						cpuUsage = float64(usage) / 1000000.0 // Convert microseconds to seconds
					}
				}
			}
		}
	}

	// Read memory usage
	memoryCurrentPath := filepath.Join(cgroupPath, "memory.current")
	if data, err := os.ReadFile(memoryCurrentPath); err == nil {
		if usage, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil {
			memoryUsed = usage
		}
	}

	return cpuUsage, memoryUsed
}

// cleanupCgroup removes the cgroup
func (s *LinuxSandbox) cleanupCgroup() error {
	if s.cgroupPath == "" {
		return nil
	}

	// Remove all processes from cgroup first
	killPath := filepath.Join(s.cgroupPath, "cgroup.kill")
	_ = os.WriteFile(killPath, []byte("1"), 0644)

	// Remove cgroup directory
	if err := os.RemoveAll(s.cgroupPath); err != nil {
		return fmt.Errorf("failed to remove cgroup: %w", err)
	}

	s.cgroupPath = ""
	return nil
}

// parseCPULimit parses CPU limit string ("1", "50%", "0.5")
func parseCPULimit(limitStr string) (float64, error) {
	limitStr = strings.TrimSpace(limitStr)
	if limitStr == "" {
		return 0, nil
	}

	// Handle percentage
	if strings.HasSuffix(limitStr, "%") {
		percentStr := strings.TrimSuffix(limitStr, "%")
		percent, err := strconv.ParseFloat(percentStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid CPU percentage: %w", err)
		}
		// Convert percentage to CPU cores (assuming 1 CPU = 100%)
		return percent / 100.0, nil
	}

	// Parse as float (number of cores)
	cores, err := strconv.ParseFloat(limitStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid CPU limit format: %w", err)
	}

	return cores, nil
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
