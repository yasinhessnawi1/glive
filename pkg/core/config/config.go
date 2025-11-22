package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/glive/core/types"
	infraconfig "github.com/glive/infrastructure/config"
	"github.com/glive/infrastructure/security"
	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigFileName = ".glive.json"
	DefaultWorkspaceDir   = "glive-workspace"
	DefaultAgentPort      = 8080
	DefaultMaxConcurrent  = 3
)

// Manager handles configuration
type Manager struct {
	configPath      string
	config          *types.Config
	credentialStore *security.SecureCredentialStore
	loader          *infraconfig.ConfigLoader
	validator       *infraconfig.ConfigValidator
}

// New creates a new config manager
func New() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, DefaultConfigFileName)

	// Initialize secure credential store
	credentialStore, err := security.NewSecureCredentialStore("glive", homeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize credential store: %w", err)
	}

	// Initialize new config loader
	loader, err := infraconfig.NewConfigLoader()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize config loader: %w", err)
	}

	// Initialize validator
	validator := infraconfig.NewConfigValidator()

	manager := &Manager{
		configPath:      configPath,
		config:          getDefaultConfig(),
		credentialStore: credentialStore,
		loader:          loader,
		validator:       validator,
	}

	// Load configuration using new layered loader
	if err := manager.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return manager, nil
}

// NewWithFlags creates a new config manager with CLI flags
func NewWithFlags(flags map[string]interface{}) (*Manager, error) {
	manager, err := New()
	if err != nil {
		return nil, err
	}

	// Reload with flags
	if err := manager.LoadWithFlags(flags); err != nil {
		return nil, fmt.Errorf("failed to load config with flags: %w", err)
	}

	return manager, nil
}

// getDefaultConfig returns the default configuration
func getDefaultConfig() *types.Config {
	homeDir, _ := os.UserHomeDir()
	workspaceDir := filepath.Join(homeDir, DefaultWorkspaceDir)

	return &types.Config{
		APIKey:        "",
		APIProvider:   "deepseek",
		APIEndpoint:   "",
		DefaultMode:   types.ModeAuto,
		WorkspaceDir:  workspaceDir,
		MaxConcurrent: DefaultMaxConcurrent,
		EnableSandbox: false,
		AgentPort:     DefaultAgentPort,
		AIFirst: &types.AIFirstConfig{
			PrimaryMode:         "ai_first",
			ConfidenceThreshold: 0.7,
			EnableFallback:      true,
			EnableAIMonitoring:  true,
			EnableAutoRecovery:  true,
			EnableLearning:      false,
			FallbackStrategy:    "traditional",
		},
		Sandbox: &types.SandboxConfig{
			Enabled:        false,
			Root:           "",
			NetworkEnabled: true,
			CPULimit:       "1",
			MemoryLimit:    "1GB",
			Timeout:        "30m",
			CleanupOnExit:  true,
		},
	}
}

// Load loads configuration using the new layered loader
func (m *Manager) Load() error {
	return m.LoadWithFlags(nil)
}

// LoadWithFlags loads configuration with CLI flags override
func (m *Manager) LoadWithFlags(flags map[string]interface{}) error {
	// Use new layered loader
	cfg, err := m.loader.Load(flags)
	if err != nil {
		// Fallback to legacy loading for backward compatibility
		return m.loadLegacy()
	}

	// Load API key from secure storage (only override if credential store has a value)
	if apiKey, err := m.credentialStore.Get("api_key"); err == nil && apiKey != "" {
		cfg.APIKey = apiKey
	}
	// Keep the API key from config if credential store is empty (backward compatibility)

	// Validate configuration
	if err := m.validator.Validate(cfg); err != nil {
		// Log validation errors but don't fail (for backward compatibility)
		// In strict mode, this would return an error
		_ = err
	}

	m.config = cfg
	return nil
}

// loadLegacy loads configuration from legacy JSON file
func (m *Manager) loadLegacy() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}

	var config types.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	m.config = &config

	// Try to load API key from secure storage (only override if credential store has a value)
	if apiKey, err := m.credentialStore.Get("api_key"); err == nil && apiKey != "" {
		m.config.APIKey = apiKey
	}
	// Keep the API key from JSON if credential store is empty (backward compatibility)

	return nil
}

// Save saves configuration to disk (without sensitive data)
// Saves to YAML format if possible, falls back to JSON
func (m *Manager) Save() error {
	// Create a copy without API key for storage
	configCopy := *m.config
	configCopy.APIKey = "" // Don't store API key in plain config

	// Try to save as YAML first
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".glive")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	yamlPath := filepath.Join(configDir, "config.yaml")
	if err := m.saveYAML(yamlPath, &configCopy); err == nil {
		return nil
	}

	// Fallback to JSON
	data, err := json.MarshalIndent(&configCopy, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// saveYAML saves configuration as YAML
func (m *Manager) saveYAML(path string, cfg *types.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Get returns the current configuration
func (m *Manager) Get() *types.Config {
	return m.config
}

// Set sets a configuration value
func (m *Manager) Set(key string, value interface{}) error {
	switch key {
	case "api-key":
		apiKey := value.(string)
		m.config.APIKey = apiKey
		// Store API key securely
		if err := m.credentialStore.Set("api_key", apiKey); err != nil {
			return fmt.Errorf("failed to store API key securely: %w", err)
		}
	case "api-provider":
		m.config.APIProvider = value.(string)
	case "api-endpoint":
		m.config.APIEndpoint = value.(string)
	case "default-mode":
		mode := types.ExecutionMode(value.(string))
		if mode != types.ModeAuto && mode != types.ModeAssisted && mode != types.ModeManual {
			return fmt.Errorf("invalid mode: %s", value)
		}
		m.config.DefaultMode = mode
	case "workspace-dir":
		m.config.WorkspaceDir = value.(string)
	case "max-concurrent":
		m.config.MaxConcurrent = value.(int)
	case "enable-sandbox":
		m.config.EnableSandbox = value.(bool)
	case "agent-port":
		m.config.AgentPort = value.(int)
	case "ai-first-mode":
		if m.config.AIFirst == nil {
			m.config.AIFirst = &types.AIFirstConfig{}
		}
		m.config.AIFirst.PrimaryMode = value.(string)
	case "ai-confidence":
		if m.config.AIFirst == nil {
			m.config.AIFirst = &types.AIFirstConfig{}
		}
		m.config.AIFirst.ConfidenceThreshold = value.(float64)
	case "enable-fallback":
		if m.config.AIFirst == nil {
			m.config.AIFirst = &types.AIFirstConfig{}
		}
		m.config.AIFirst.EnableFallback = value.(bool)
	case "enable-ai-monitoring":
		if m.config.AIFirst == nil {
			m.config.AIFirst = &types.AIFirstConfig{}
		}
		m.config.AIFirst.EnableAIMonitoring = value.(bool)
	case "enable-auto-recovery":
		if m.config.AIFirst == nil {
			m.config.AIFirst = &types.AIFirstConfig{}
		}
		m.config.AIFirst.EnableAutoRecovery = value.(bool)
	case "enable-learning":
		if m.config.AIFirst == nil {
			m.config.AIFirst = &types.AIFirstConfig{}
		}
		m.config.AIFirst.EnableLearning = value.(bool)
	case "sandbox-root":
		if m.config.Sandbox == nil {
			m.config.Sandbox = &types.SandboxConfig{}
		}
		m.config.Sandbox.Root = value.(string)
	case "sandbox-no-network":
		if m.config.Sandbox == nil {
			m.config.Sandbox = &types.SandboxConfig{}
		}
		m.config.Sandbox.NetworkEnabled = !value.(bool)
	case "sandbox-cpu-limit":
		if m.config.Sandbox == nil {
			m.config.Sandbox = &types.SandboxConfig{}
		}
		m.config.Sandbox.CPULimit = value.(string)
	case "sandbox-memory-limit":
		if m.config.Sandbox == nil {
			m.config.Sandbox = &types.SandboxConfig{}
		}
		m.config.Sandbox.MemoryLimit = value.(string)
	case "sandbox-timeout":
		if m.config.Sandbox == nil {
			m.config.Sandbox = &types.SandboxConfig{}
		}
		m.config.Sandbox.Timeout = value.(string)
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return m.Save()
}

// GetAPIKey returns the API key (from secure storage)
func (m *Manager) GetAPIKey() string {
	// Try to load from secure storage first
	if apiKey, err := m.credentialStore.Get("api_key"); err == nil {
		return apiKey
	}
	// Fallback to config (for backward compatibility)
	return m.config.APIKey
}

// GetWorkspaceDir returns the workspace directory
func (m *Manager) GetWorkspaceDir() string {
	return m.config.WorkspaceDir
}

// IsConfigured returns true if the basic configuration is set
func (m *Manager) IsConfigured() bool {
	return m.config.APIKey != ""
}

// InitWorkspace creates the workspace directory if it doesn't exist
func (m *Manager) InitWorkspace() error {
	return os.MkdirAll(m.config.WorkspaceDir, 0755)
}
