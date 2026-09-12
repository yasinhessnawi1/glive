//go:build windows

package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// WindowsSandbox implements Sandbox interface for Windows using Job Objects
type WindowsSandbox struct {
	config      SandboxConfig
	rootPath    string
	jobHandle   windows.Handle
	initialized bool
}

// NewWindowsSandbox creates a new Windows sandbox instance
func NewWindowsSandbox() *WindowsSandbox {
	return &WindowsSandbox{}
}

// newPlatformSandbox creates a platform-specific sandbox (Windows implementation)
func newPlatformSandbox() (Sandbox, error) {
	return NewWindowsSandbox(), nil
}

// Initialize prepares the Windows sandbox environment
func (s *WindowsSandbox) Initialize(ctx context.Context, config SandboxConfig) error {
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
	workDir := filepath.Join(s.rootPath, strings.TrimPrefix(config.WorkDir, "\\"))
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}

	// Create Job Object for process containment
	jobName := fmt.Sprintf("GLiveSandbox_%d", os.Getpid())
	jobHandle, err := windows.CreateJobObject(nil, windows.StringToUTF16Ptr(jobName))
	if err != nil {
		return fmt.Errorf("failed to create job object: %w", err)
	}
	s.jobHandle = jobHandle

	// Setup mount points (on Windows, these are just directory copies or junctions)
	for _, mount := range config.MountPoints {
		if err := ValidateMountPoint(s.rootPath, mount); err != nil {
			return err
		}

		dest := filepath.Join(s.rootPath, strings.TrimPrefix(mount.Destination, "\\"))
		if err := os.MkdirAll(dest, 0755); err != nil {
			return fmt.Errorf("failed to create mount point: %w", err)
		}

		// NOTE: read-only mounts are NOT enforced on Windows. The mount point is
		// created read-write regardless of mount.ReadOnly. Enforcing it would mean
		// a directory junction (which needs admin or developer mode) or an ACL on
		// the destination. This was an empty `if mount.ReadOnly && ...` branch,
		// which read as though the case were handled; it never was. The Mode-B
		// isolation work owns closing it - see DEFERRED_WORK_AUDIT.
	}

	s.initialized = true
	return nil
}

// Execute runs a command in the Windows sandbox
func (s *WindowsSandbox) Execute(ctx context.Context, cmd Command) (*Result, error) {
	if !s.initialized {
		return nil, fmt.Errorf("sandbox not initialized")
	}

	startTime := time.Now()

	// Parse resource limits
	cpuLimit, err := parseCPULimit(s.config.CPULimit)
	if err != nil {
		return nil, fmt.Errorf("invalid CPU limit: %w", err)
	}

	memoryLimit, err := parseMemoryLimit(s.config.MemoryLimit)
	if err != nil {
		return nil, fmt.Errorf("invalid memory limit: %w", err)
	}

	// Apply Job Object limits
	if err := s.applyJobLimits(cpuLimit, memoryLimit); err != nil {
		return nil, fmt.Errorf("failed to apply job limits: %w", err)
	}

	// Build command
	execCmd := s.buildCommand(ctx, cmd)

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

	// Start command
	if err := execCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Assign process to job object
	processHandle, _ := windows.OpenProcess(windows.PROCESS_ALL_ACCESS, false, uint32(execCmd.Process.Pid))
	if processHandle != 0 {
		defer windows.CloseHandle(processHandle)
		if assignErr := windows.AssignProcessToJobObject(s.jobHandle, processHandle); assignErr != nil {
			execCmd.Process.Kill()
			return nil, fmt.Errorf("failed to assign process to job: %w", assignErr)
		}
	}

	// Wait for command completion with timeout
	done := make(chan error, 1)
	go func() {
		done <- execCmd.Wait()
	}()

	var execErr error
	select {
	case <-timeoutCtx.Done():
		// Kill process on timeout
		execCmd.Process.Kill()
		execErr = timeoutCtx.Err()
	case execErr = <-done:
	}

	duration := time.Since(startTime)

	exitCode := 0
	if execErr != nil {
		exitErr := &exec.ExitError{}
		if errors.As(execErr, &exitErr) {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		}
	}

	// Get resource usage from job object
	cpuUsage, memoryUsed := s.getJobResourceUsage()

	return &Result{
		ExitCode:   exitCode,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		Success:    exitCode == 0 && execErr == nil,
		Error:      execErr,
		Duration:   duration,
		CPUUsage:   cpuUsage,
		MemoryUsed: memoryUsed,
	}, nil
}

// Cleanup removes sandbox artifacts
func (s *WindowsSandbox) Cleanup() error {
	if s.jobHandle != 0 {
		// Terminate all processes in job
		windows.TerminateJobObject(s.jobHandle, 1)
		windows.CloseHandle(s.jobHandle)
		s.jobHandle = 0
	}

	// Remove sandbox directory
	if s.rootPath != "" {
		return os.RemoveAll(s.rootPath)
	}

	return nil
}

// buildCommand builds the command for execution
func (s *WindowsSandbox) buildCommand(ctx context.Context, cmd Command) *exec.Cmd {
	// Set working directory
	workDir := filepath.Join(s.rootPath, strings.TrimPrefix(cmd.WorkingDir, "\\"))
	if workDir == "" || cmd.WorkingDir == "" {
		workDir = filepath.Join(s.rootPath, strings.TrimPrefix(s.config.WorkDir, "\\"))
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

	return execCmd
}

// Windows Job Object constants (not all are in golang.org/x/sys/windows)
const (
	JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE  = 0x00002000
	JOB_OBJECT_LIMIT_PROCESS_MEMORY     = 0x00000100
	JOB_OBJECT_LIMIT_JOB_MEMORY         = 0x00000200
	JobObjectBasicLimitInformation      = 2
	JobObjectExtendedLimitInformation   = 9
	JobObjectBasicAccountingInformation = 1
)

// JOBOBJECT_BASIC_LIMIT_INFORMATION for setting limits
type jobObjectBasicLimitInfo struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

// JOBOBJECT_EXTENDED_LIMIT_INFORMATION for setting extended limits
type jobObjectExtendedLimitInfo struct {
	BasicLimitInformation jobObjectBasicLimitInfo
	IoInfo                [6]uint64 // IO_COUNTERS - simplified
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

// applyJobLimits applies CPU and memory limits to the job object
func (s *WindowsSandbox) applyJobLimits(cpuLimit float64, memoryLimit int64) error {
	// Set memory limit and kill on job close
	extendedLimitInfo := jobObjectExtendedLimitInfo{
		BasicLimitInformation: jobObjectBasicLimitInfo{
			LimitFlags: JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}

	if memoryLimit > 0 {
		extendedLimitInfo.BasicLimitInformation.LimitFlags |= JOB_OBJECT_LIMIT_PROCESS_MEMORY | JOB_OBJECT_LIMIT_JOB_MEMORY
		extendedLimitInfo.ProcessMemoryLimit = uintptr(memoryLimit)
		extendedLimitInfo.JobMemoryLimit = uintptr(memoryLimit)
	}

	_, err := setInformationJobObject(
		s.jobHandle,
		JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&extendedLimitInfo)),
		uint32(unsafe.Sizeof(extendedLimitInfo)),
	)
	if err != nil {
		return fmt.Errorf("failed to set job limits: %w", err)
	}

	// Note: CPU rate limiting requires Windows 8+ and specific API calls
	// that are complex to implement. For now, we only support memory limits.
	_ = cpuLimit

	return nil
}

// setInformationJobObject wrapper for the Windows API
func setInformationJobObject(job windows.Handle, infoClass uint32, info uintptr, infoLen uint32) (bool, error) {
	r1, _, err := syscall.NewLazyDLL("kernel32.dll").NewProc("SetInformationJobObject").Call(
		uintptr(job),
		uintptr(infoClass),
		info,
		uintptr(infoLen),
	)
	if r1 == 0 {
		return false, err
	}
	return true, nil
}

// jobObjectBasicAccountingInfo for querying resource usage
type jobObjectBasicAccountingInfo struct {
	TotalUserTime             int64
	TotalKernelTime           int64
	ThisPeriodTotalUserTime   int64
	ThisPeriodTotalKernelTime int64
	TotalPageFaultCount       uint32
	TotalProcesses            uint32
	ActiveProcesses           uint32
	TotalTerminatedProcesses  uint32
}

// getJobResourceUsage reads resource usage from job object
func (s *WindowsSandbox) getJobResourceUsage() (float64, int64) {
	var cpuUsage float64
	var memoryUsed int64

	// Query job object for resource usage
	basicAccInfo := jobObjectBasicAccountingInfo{}
	var returnLength uint32

	r1, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("QueryInformationJobObject").Call(
		uintptr(s.jobHandle),
		uintptr(JobObjectBasicAccountingInformation),
		uintptr(unsafe.Pointer(&basicAccInfo)),
		uintptr(unsafe.Sizeof(basicAccInfo)),
		uintptr(unsafe.Pointer(&returnLength)),
	)

	if r1 != 0 {
		// Calculate CPU usage (TotalUserTime + TotalKernelTime)
		totalTime := basicAccInfo.TotalUserTime + basicAccInfo.TotalKernelTime
		cpuUsage = float64(totalTime) / 10000000.0                  // Convert 100-nanosecond intervals to seconds
		memoryUsed = int64(basicAccInfo.TotalPageFaultCount) * 4096 // Approximate
	}

	return cpuUsage, memoryUsed
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
