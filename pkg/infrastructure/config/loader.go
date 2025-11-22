package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glive/core/types"
)

const (
	// DefaultConfigDir is the default configuration directory
	DefaultConfigDir = ".glive"
	// UserConfigFileName is the user config file name
	UserConfigFileName = "config.yaml"
	// ProjectConfigFileName is the project config file name
	ProjectConfigFileName = ".glive.yaml"
	// LegacyConfigFileName is the legacy JSON config file name
	LegacyConfigFileName = ".glive.json"
)

// ConfigLoader handles layered configuration loading
type ConfigLoader struct {
	defaults    map[string]interface{}
	projectFile string
	userFile    string
	legacyFile  string
	envPrefix   string
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader() (*ConfigLoader, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, DefaultConfigDir)
	userFile := filepath.Join(configDir, UserConfigFileName)
	legacyFile := filepath.Join(homeDir, LegacyConfigFileName)

	// Find project config file (current directory or parent directories)
	projectFile := findProjectConfig()

	// Get default configuration
	defaults := getDefaultConfigMap()

	return &ConfigLoader{
		defaults:    defaults,
		projectFile: projectFile,
		userFile:    userFile,
		legacyFile:  legacyFile,
		envPrefix:   "GLIVE_",
	}, nil
}

// Load loads configuration with proper layering
// Priority (highest to lowest):
// 1. Command-line flags (passed via ApplyFlags)
// 2. Environment variables
// 3. User config file (~/.glive/config.yaml or ~/.glive.json)
// 4. Project config file (.glive.yaml)
// 5. Default values
func (l *ConfigLoader) Load(flags map[string]interface{}) (*types.Config, error) {
	// Start with defaults
	cfg := make(map[string]interface{})
	for k, v := range l.defaults {
		cfg[k] = v
	}

	// Layer 4: Project config file
	if l.projectFile != "" {
		if projectCfg, err := l.loadFile(l.projectFile); err == nil {
			cfg = merge(cfg, projectCfg)
		}
	}

	// Layer 3: User config file (try YAML first, then JSON)
	if userCfg, err := l.loadFile(l.userFile); err == nil {
		cfg = merge(cfg, userCfg)
	} else if legacyCfg, err := l.loadFile(l.legacyFile); err == nil {
		cfg = merge(cfg, legacyCfg)
	}

	// Layer 2: Environment variables
	cfg = l.applyEnv(cfg)

	// Layer 1: Command-line flags (highest priority)
	if flags != nil {
		cfg = merge(cfg, flags)
	}

	// Convert map to Config struct
	return l.mapToConfig(cfg)
}

// loadFile loads a configuration file (YAML or JSON)
func (l *ConfigLoader) loadFile(filePath string) (map[string]interface{}, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path is empty")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Try YAML first
	if cfg, err := parseYAML(data); err == nil {
		return cfg, nil
	}

	// Fallback to JSON
	if cfg, err := parseJSON(data); err == nil {
		return cfg, nil
	}

	return nil, fmt.Errorf("failed to parse config file: %s", filePath)
}

// merge merges two configuration maps, with src overriding dst
func merge(dst, src map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Copy dst
	for k, v := range dst {
		result[k] = v
	}
	
	// Override with src
	for k, v := range src {
		result[k] = v
	}
	
	return result
}

// getDefaultConfigMap returns default configuration as a map
func getDefaultConfigMap() map[string]interface{} {
	homeDir, _ := os.UserHomeDir()
	workspaceDir := filepath.Join(homeDir, "glive-workspace")

	return map[string]interface{}{
		"api-key":        "",
		"api-provider":   "deepseek",
		"api-endpoint":   "",
		"default-mode":   "auto",
		"workspace-dir":  workspaceDir,
		"max-concurrent": 3,
		"enable-sandbox": false,
		"agent-port":     8080,
		"log-level":      "info",
		"log-file":       filepath.Join(homeDir, DefaultConfigDir, "glive.log"),
	}
}

// mapToConfig converts a configuration map to a Config struct
func (l *ConfigLoader) mapToConfig(cfg map[string]interface{}) (*types.Config, error) {
	result := &types.Config{}

	// Helper to get string value
	getString := func(key string) string {
		if val, ok := cfg[key]; ok {
			if str, ok := val.(string); ok {
				return str
			}
		}
		return ""
	}

	// Helper to get int value
	getInt := func(key string, defaultValue int) int {
		if val, ok := cfg[key]; ok {
			switch v := val.(type) {
			case int:
				return v
			case int64:
				return int(v)
			case float64:
				return int(v)
			case string:
				// Try to parse string
				var i int
				if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
					return i
				}
			}
		}
		return defaultValue
	}

	// Helper to get bool value
	getBool := func(key string, defaultValue bool) bool {
		if val, ok := cfg[key]; ok {
			switch v := val.(type) {
			case bool:
				return v
			case string:
				return v == "true" || v == "1" || v == "yes"
			case int:
				return v != 0
			}
		}
		return defaultValue
	}

	result.APIKey = getString("api-key")
	result.APIProvider = getString("api-provider")
	result.APIEndpoint = getString("api-endpoint")
	result.DefaultMode = types.ExecutionMode(getString("default-mode"))
	result.WorkspaceDir = getString("workspace-dir")
	result.MaxConcurrent = getInt("max-concurrent", 3)
	result.EnableSandbox = getBool("enable-sandbox", false)
	result.AgentPort = getInt("agent-port", 8080)

	// Parse sandbox configuration
	if sandboxVal, ok := cfg["sandbox"]; ok {
		if sandboxMap, ok := sandboxVal.(map[string]interface{}); ok {
			result.Sandbox = &types.SandboxConfig{
				Enabled:        getBoolFromMap(sandboxMap, "enabled", false),
				Root:           getStringFromMap(sandboxMap, "root"),
				NetworkEnabled: getBoolFromMap(sandboxMap, "network_enabled", true),
				CPULimit:       getStringFromMap(sandboxMap, "cpu_limit"),
				MemoryLimit:    getStringFromMap(sandboxMap, "memory_limit"),
				Timeout:        getStringFromMap(sandboxMap, "timeout"),
				CleanupOnExit:  getBoolFromMap(sandboxMap, "cleanup_on_exit", true),
			}
		}
	}

	// Apply sandbox flags if set
	if result.Sandbox == nil {
		result.Sandbox = &types.SandboxConfig{}
	}
	if sandboxRoot := getString("sandbox-root"); sandboxRoot != "" {
		result.Sandbox.Root = sandboxRoot
	}
	if getBool("sandbox-no-network", false) {
		result.Sandbox.NetworkEnabled = false
	}
	if cpuLimit := getString("sandbox-cpu-limit"); cpuLimit != "" {
		result.Sandbox.CPULimit = cpuLimit
	}
	if memoryLimit := getString("sandbox-memory-limit"); memoryLimit != "" {
		result.Sandbox.MemoryLimit = memoryLimit
	}
	if timeout := getString("sandbox-timeout"); timeout != "" {
		result.Sandbox.Timeout = timeout
	}
	if getBool("enable-sandbox", false) {
		result.Sandbox.Enabled = true
	}

	return result, nil
}

// Helper functions for nested map access
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getBoolFromMap(m map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case bool:
			return v
		case string:
			return v == "true" || v == "1" || v == "yes"
		case int:
			return v != 0
		}
	}
	return defaultValue
}

// findProjectConfig searches for project config file in current and parent directories
func findProjectConfig() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	dir := wd
	for {
		configPath := filepath.Join(dir, ProjectConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached root
		}
		dir = parent
	}

	return ""
}

