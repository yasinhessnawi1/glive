package ai

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/glive/core/types"
)

// RecoveryStrategy defines an interface for error recovery strategies
type RecoveryStrategy interface {
	Name() string
	Description() string
	CanHandle(ctx *ErrorContext) bool
	Apply(cmd *types.Command, output, errorMsg string) (*types.Command, string, error)
	Priority() int
	RequiresApproval() bool
}

// LintingBypassStrategy skips linting for builds
type LintingBypassStrategy struct{}

func (s *LintingBypassStrategy) Name() string { return "Linting Bypass" }
func (s *LintingBypassStrategy) Description() string {
	return "Skip linting checks during build"
}
func (s *LintingBypassStrategy) Priority() int          { return 1 }
func (s *LintingBypassStrategy) RequiresApproval() bool { return false }

func (s *LintingBypassStrategy) CanHandle(ctx *ErrorContext) bool {
	return ctx.Category == ErrorCategoryLinting
}

func (s *LintingBypassStrategy) Apply(cmd *types.Command, output, errorMsg string) (*types.Command, string, error) {
	fixedCmd := *cmd

	// Detect build commands and add no-lint flags
	if strings.Contains(cmd.Command, "build") {
		if strings.Contains(cmd.Command, "next build") {
			// Next.js: set env var to skip linting
			if fixedCmd.Env == nil {
				fixedCmd.Env = make(map[string]string)
			}
			fixedCmd.Env["SKIP_LINT"] = "true"
			fixedCmd.Env["ESLINT_NO_DEV_ERRORS"] = "true"
			return &fixedCmd, "Added SKIP_LINT=true to bypass linting errors", nil
		}

		if strings.Contains(cmd.Command, "npm") || strings.Contains(cmd.Command, "pnpm") || strings.Contains(cmd.Command, "yarn") {
			// Add -- --no-lint flag
			fixedCmd.Command = cmd.Command + " -- --no-lint"
			return &fixedCmd, "Added --no-lint flag to skip linting", nil
		}
	}

	return nil, "", fmt.Errorf("cannot apply linting bypass to this command")
}

// DevModeFallbackStrategy uses dev mode instead of production
type DevModeFallbackStrategy struct{}

func (s *DevModeFallbackStrategy) Name() string { return "Dev Mode Fallback" }
func (s *DevModeFallbackStrategy) Description() string {
	return "Use development mode instead of production build"
}
func (s *DevModeFallbackStrategy) Priority() int          { return 2 }
func (s *DevModeFallbackStrategy) RequiresApproval() bool { return false }

func (s *DevModeFallbackStrategy) CanHandle(ctx *ErrorContext) bool {
	return ctx.Category == ErrorCategoryBuildFailure || ctx.Category == ErrorCategoryLinting
}

func (s *DevModeFallbackStrategy) Apply(cmd *types.Command, output, errorMsg string) (*types.Command, string, error) {
	fixedCmd := *cmd

	// Replace build/start commands with dev
	replacements := map[string]string{
		"npm run build": "npm run dev",
		"npm start":     "npm run dev",
		"pnpm build":    "pnpm dev",
		"pnpm start":    "pnpm dev",
		"yarn build":    "yarn dev",
		"yarn start":    "yarn dev",
		"next build":    "next dev",
		"next start":    "next dev",
	}

	for old, new := range replacements {
		if strings.Contains(cmd.Command, old) {
			fixedCmd.Command = strings.Replace(cmd.Command, old, new, 1)
			return &fixedCmd, fmt.Sprintf("Switched from '%s' to '%s' to avoid build issues", old, new), nil
		}
	}

	return nil, "", fmt.Errorf("cannot apply dev mode fallback to this command")
}

// PortConflictStrategy changes ports automatically
type PortConflictStrategy struct{}

func (s *PortConflictStrategy) Name() string { return "Port Conflict Resolution" }
func (s *PortConflictStrategy) Description() string {
	return "Change port to avoid conflicts"
}
func (s *PortConflictStrategy) Priority() int          { return 3 }
func (s *PortConflictStrategy) RequiresApproval() bool { return false }

func (s *PortConflictStrategy) CanHandle(ctx *ErrorContext) bool {
	return ctx.Category == ErrorCategoryPortConflict
}

func (s *PortConflictStrategy) Apply(cmd *types.Command, output, errorMsg string) (*types.Command, string, error) {
	fixedCmd := *cmd

	// Extract current port from error message
	portRegex := regexp.MustCompile(`(?:port\s+|:)(\d{4,5})`)
	matches := portRegex.FindStringSubmatch(output + errorMsg)

	currentPort := 3000 // default
	if len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &currentPort)
	}

	// Increment port
	newPort := currentPort + 1

	// Set PORT environment variable
	if fixedCmd.Env == nil {
		fixedCmd.Env = make(map[string]string)
	}
	fixedCmd.Env["PORT"] = fmt.Sprintf("%d", newPort)

	return &fixedCmd, fmt.Sprintf("Changed port from %d to %d", currentPort, newPort), nil
}

// MissingDependencyStrategy installs missing packages
type MissingDependencyStrategy struct{}

func (s *MissingDependencyStrategy) Name() string { return "Missing Dependency Install" }
func (s *MissingDependencyStrategy) Description() string {
	return "Install missing dependencies"
}
func (s *MissingDependencyStrategy) Priority() int          { return 4 }
func (s *MissingDependencyStrategy) RequiresApproval() bool { return false }

func (s *MissingDependencyStrategy) CanHandle(ctx *ErrorContext) bool {
	return ctx.Category == ErrorCategoryMissingDep
}

func (s *MissingDependencyStrategy) Apply(cmd *types.Command, output, errorMsg string) (*types.Command, string, error) {
	combinedText := output + "\n" + errorMsg

	// Extract module name from error
	patterns := []string{
		`Cannot find module ['"]([^'"]+)['"]`,
		`ModuleNotFoundError: No module named ['"]([^'"]+)['"]`,
		`Error: Cannot find module '([^']+)'`,
	}

	var moduleName string
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(combinedText)
		if len(matches) > 1 {
			moduleName = matches[1]
			break
		}
	}

	if moduleName == "" {
		return nil, "", fmt.Errorf("could not extract module name from error")
	}

	// Determine package manager from original command
	var installCmd string
	if strings.Contains(cmd.Command, "npm") {
		installCmd = fmt.Sprintf("npm install %s", moduleName)
	} else if strings.Contains(cmd.Command, "pnpm") {
		installCmd = fmt.Sprintf("pnpm install %s", moduleName)
	} else if strings.Contains(cmd.Command, "yarn") {
		installCmd = fmt.Sprintf("yarn add %s", moduleName)
	} else if strings.Contains(cmd.Command, "python") || strings.Contains(cmd.Command, "pip") {
		installCmd = fmt.Sprintf("pip install %s", moduleName)
	} else {
		installCmd = fmt.Sprintf("npm install %s", moduleName)
	}

	fixedCmd := &types.Command{
		ID:          cmd.ID + "-install",
		Description: fmt.Sprintf("Install missing dependency: %s", moduleName),
		Command:     installCmd,
		WorkingDir:  cmd.WorkingDir,
		Stage:       "setup",
		Required:    true,
		Status:      types.CommandPending,
		Env:         cmd.Env,
	}

	return fixedCmd, fmt.Sprintf("Installing missing module '%s'", moduleName), nil
}

// NetworkRetryStrategy retries network failures
type NetworkRetryStrategy struct{}

func (s *NetworkRetryStrategy) Name() string { return "Network Retry" }
func (s *NetworkRetryStrategy) Description() string {
	return "Retry command after network failure"
}
func (s *NetworkRetryStrategy) Priority() int          { return 5 }
func (s *NetworkRetryStrategy) RequiresApproval() bool { return false }

func (s *NetworkRetryStrategy) CanHandle(ctx *ErrorContext) bool {
	return ctx.Category == ErrorCategoryNetwork
}

func (s *NetworkRetryStrategy) Apply(cmd *types.Command, output, errorMsg string) (*types.Command, string, error) {
	// Simply return the same command for retry
	fixedCmd := *cmd
	fixedCmd.Status = types.CommandPending

	return &fixedCmd, "Retrying after network error", nil
}

// EnvFileRecoveryStrategy handles .env file creation issues
type EnvFileRecoveryStrategy struct{}

func (s *EnvFileRecoveryStrategy) Name() string { return "Environment File Recovery" }
func (s *EnvFileRecoveryStrategy) Description() string {
	return "Handle missing .env files by reading .env.example or creating minimal config"
}
func (s *EnvFileRecoveryStrategy) Priority() int          { return 1 }
func (s *EnvFileRecoveryStrategy) RequiresApproval() bool { return false }

func (s *EnvFileRecoveryStrategy) CanHandle(ctx *ErrorContext) bool {
	// Check if error is related to env file operations
	return ctx.Category == ErrorCategoryRuntime || ctx.Category == ErrorCategoryUnknown
}

func (s *EnvFileRecoveryStrategy) Apply(cmd *types.Command, output, errorMsg string) (*types.Command, string, error) {
	combinedText := strings.ToLower(output + "\n" + errorMsg)

	// Check if this is an env file related error
	envKeywords := []string{".env", "env.example", "copy", "cp", "environment"}
	hasEnvError := false
	for _, keyword := range envKeywords {
		if strings.Contains(combinedText, keyword) {
			hasEnvError = true
			break
		}
	}

	if !hasEnvError {
		return nil, "", fmt.Errorf("not an env file error")
	}

	// Strategy 1: Try to read .env.example and create .env.local
	// Strategy 2: Try multiple copy command variations (cp, copy, powershell)
	// Strategy 3: Create empty .env.local file
	// Strategy 4: Skip env file entirely

	// Check if command was a copy operation
	if strings.Contains(cmd.Command, "copy") || strings.Contains(cmd.Command, "cp") {
		// Try OS-specific copy commands
		var newCmd string
		var explanation string

		// Determine working directory
		workDir := cmd.WorkingDir
		if workDir == "" {
			workDir = "."
		}

		// Try PowerShell on Windows, cat on Unix
		if strings.Contains(cmd.Command, "copy") {
			// Windows - try multiple approaches
			newCmd = fmt.Sprintf(`if exist "%s\\.env.example" (powershell -Command "Copy-Item '%s\\.env.example' '%s\\.env.local'") else (echo # Auto-generated minimal .env.local > "%s\\.env.local")`,
				workDir, workDir, workDir, workDir)
			explanation = "Using PowerShell to copy .env.example, or creating minimal .env.local if example doesn't exist"
		} else {
			// Unix - try cat with fallback
			newCmd = fmt.Sprintf(`if [ -f "%s/.env.example" ]; then cat "%s/.env.example" > "%s/.env.local"; else echo "# Auto-generated minimal .env.local" > "%s/.env.local"; fi`,
				workDir, workDir, workDir, workDir)
			explanation = "Using cat to copy .env.example, or creating minimal .env.local if example doesn't exist"
		}

		fixedCmd := &types.Command{
			ID:          cmd.ID + "-env-recovery",
			Description: "Recover from .env file operation failure",
			Command:     newCmd,
			WorkingDir:  cmd.WorkingDir,
			Stage:       cmd.Stage,
			Required:    false, // Not required - we can skip env files
			Status:      types.CommandPending,
			Env:         cmd.Env,
		}

		return fixedCmd, explanation, nil
	}

	return nil, "", fmt.Errorf("cannot apply env file recovery to this command")
}

// SkipNonCriticalCommandStrategy skips non-critical commands that fail
type SkipNonCriticalCommandStrategy struct{}

func (s *SkipNonCriticalCommandStrategy) Name() string { return "Skip Non-Critical Command" }
func (s *SkipNonCriticalCommandStrategy) Description() string {
	return "Skip optional commands that fail (like env file setup)"
}
func (s *SkipNonCriticalCommandStrategy) Priority() int          { return 10 }
func (s *SkipNonCriticalCommandStrategy) RequiresApproval() bool { return false }

func (s *SkipNonCriticalCommandStrategy) CanHandle(ctx *ErrorContext) bool {
	// This is a last resort strategy
	return true
}

func (s *SkipNonCriticalCommandStrategy) Apply(cmd *types.Command, output, errorMsg string) (*types.Command, string, error) {
	// Check if command is related to optional setup tasks
	optionalPatterns := []string{
		"copy",
		"cp",
		".env",
		"config",
		"setup",
	}

	cmdLower := strings.ToLower(cmd.Command)
	isOptional := false
	for _, pattern := range optionalPatterns {
		if strings.Contains(cmdLower, pattern) {
			isOptional = true
			break
		}
	}

	if isOptional && !cmd.Required {
		// Return nil to signal that we should skip this command
		return nil, "Skipping optional command - project can run without it", nil
	}

	return nil, "", fmt.Errorf("command is critical and cannot be skipped")
}

// GetAllStrategies returns all available recovery strategies
func GetAllStrategies() []RecoveryStrategy {
	return []RecoveryStrategy{
		&EnvFileRecoveryStrategy{},      // Priority 1
		&LintingBypassStrategy{},         // Priority 1
		&DevModeFallbackStrategy{},       // Priority 2
		&PortConflictStrategy{},          // Priority 3
		&MissingDependencyStrategy{},     // Priority 4
		&NetworkRetryStrategy{},          // Priority 5
		&SkipNonCriticalCommandStrategy{}, // Priority 10 (last resort)
	}
}

// SelectStrategies returns applicable strategies for the given error context
func SelectStrategies(ctx *ErrorContext) []RecoveryStrategy {
	allStrategies := GetAllStrategies()
	var applicable []RecoveryStrategy

	for _, strategy := range allStrategies {
		if strategy.CanHandle(ctx) {
			applicable = append(applicable, strategy)
		}
	}

	// Sort by priority (lower number = higher priority)
	// Already sorted by priority in GetAllStrategies

	return applicable
}
