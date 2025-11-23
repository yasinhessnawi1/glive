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

// GetAllStrategies returns all available recovery strategies
func GetAllStrategies() []RecoveryStrategy {
	return []RecoveryStrategy{
		&LintingBypassStrategy{},
		&DevModeFallbackStrategy{},
		&PortConflictStrategy{},
		&MissingDependencyStrategy{},
		&NetworkRetryStrategy{},
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
