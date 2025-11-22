package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/glive/core/types"
)

var (
	// ErrValidationFailed indicates configuration validation failed
	ErrValidationFailed = errors.New("configuration validation failed")
)

// ConfigValidator validates configuration
type ConfigValidator struct {
	rules []ValidationRule
}

// ValidationRule defines a validation rule
type ValidationRule func(*types.Config) error

// NewConfigValidator creates a new configuration validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		rules: []ValidationRule{
			validateAPIKeyRequired,
			validatePortRange,
			validateWorkspaceDir,
			validateMode,
			validateProvider,
			validateMaxConcurrent,
		},
	}
}

// Validate validates the configuration
func (v *ConfigValidator) Validate(cfg *types.Config) error {
	var errs []error

	for _, rule := range v.rules {
		if err := rule(cfg); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%w: %v", ErrValidationFailed, errors.Join(errs...))
	}

	return nil
}

// validateAPIKeyRequired checks if API key is required
func validateAPIKeyRequired(cfg *types.Config) error {
	if cfg.DefaultMode == types.ModeAuto && cfg.APIKey == "" {
		return fmt.Errorf("api-key is required when default-mode is 'auto'")
	}
	return nil
}

// validatePortRange validates agent port is in valid range
func validatePortRange(cfg *types.Config) error {
	if cfg.AgentPort < 1024 || cfg.AgentPort > 65535 {
		return fmt.Errorf("agent-port must be between 1024 and 65535, got %d", cfg.AgentPort)
	}
	return nil
}

// validateWorkspaceDir validates workspace directory exists or can be created
func validateWorkspaceDir(cfg *types.Config) error {
	if cfg.WorkspaceDir == "" {
		return fmt.Errorf("workspace-dir cannot be empty")
	}

	// Expand ~ to home directory
	expandedDir := cfg.WorkspaceDir
	if strings.HasPrefix(expandedDir, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		expandedDir = filepath.Join(homeDir, strings.TrimPrefix(expandedDir, "~"))
	}

	// Check if directory exists
	if info, err := os.Stat(expandedDir); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("workspace-dir exists but is not a directory: %s", expandedDir)
		}
		return nil
	}

	// Check if parent directory exists (so we can create it)
	parentDir := filepath.Dir(expandedDir)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		return fmt.Errorf("workspace-dir parent directory does not exist: %s", parentDir)
	}

	return nil
}

// validateMode validates execution mode
func validateMode(cfg *types.Config) error {
	validModes := map[types.ExecutionMode]bool{
		types.ModeAuto:     true,
		types.ModeAssisted: true,
		types.ModeManual:   true,
	}

	if !validModes[cfg.DefaultMode] {
		return fmt.Errorf("default-mode must be one of: auto, assisted, manual, got: %s", cfg.DefaultMode)
	}

	return nil
}

// validateProvider validates API provider
func validateProvider(cfg *types.Config) error {
	validProviders := map[string]bool{
		"deepseek": true,
		"openai":   true,
		"claude":   true,
		"ollama":   true,
		"local":    true,
	}

	if !validProviders[cfg.APIProvider] {
		return fmt.Errorf("api-provider must be one of: deepseek, openai, claude, ollama, local, got: %s", cfg.APIProvider)
	}

	return nil
}

// validateMaxConcurrent validates max concurrent value
func validateMaxConcurrent(cfg *types.Config) error {
	if cfg.MaxConcurrent <= 0 {
		return fmt.Errorf("max-concurrent must be greater than 0, got %d", cfg.MaxConcurrent)
	}

	if cfg.MaxConcurrent > 100 {
		return fmt.Errorf("max-concurrent should not exceed 100, got %d", cfg.MaxConcurrent)
	}

	return nil
}

