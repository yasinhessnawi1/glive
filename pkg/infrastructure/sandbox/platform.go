package sandbox

import (
	"os"
	"os/exec"
	"runtime"
)

// NewSandbox creates a platform-specific sandbox instance
func NewSandbox() (Sandbox, error) {
	return newPlatformSandbox()
}

// CheckSandboxAvailable checks if sandboxing is available on this platform
func CheckSandboxAvailable() bool {
	switch runtime.GOOS {
	case "linux":
		// Check for cgroup v2 support
		return checkLinuxSandboxAvailable()
	case "windows":
		// Job Objects are always available on Windows
		return true
	case "darwin":
		// Check for sandbox-exec or fallback to resource limits
		return checkMacOSSandboxAvailable()
	default:
		return false
	}
}

// checkLinuxSandboxAvailable checks if Linux sandbox features are available
func checkLinuxSandboxAvailable() bool {
	// Check for cgroup v2
	if _, err := os.Stat("/sys/fs/cgroup"); err == nil {
		return true
	}
	return false
}

// checkMacOSSandboxAvailable checks if macOS sandbox features are available
func checkMacOSSandboxAvailable() bool {
	// sandbox-exec may not be available, but we can still use resource limits
	if runtime.GOOS == "darwin" {
		_, err := exec.LookPath("sandbox-exec")
		// Return true even if sandbox-exec not found - we can use resource limits
		return err == nil || true
	}
	return false
}

