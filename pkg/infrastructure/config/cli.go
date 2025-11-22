package config

// CLIFlags represents command-line flags that can override configuration
type CLIFlags struct {
	APIKey        *string
	APIProvider   *string
	APIEndpoint   *string
	DefaultMode   *string
	WorkspaceDir  *string
	MaxConcurrent *int
	EnableSandbox *bool
	AgentPort     *int
	LogLevel      *string
	LogFile       *string
}

// ToMap converts CLI flags to a configuration map
func (f *CLIFlags) ToMap() map[string]interface{} {
	cfg := make(map[string]interface{})

	if f.APIKey != nil {
		cfg["api-key"] = *f.APIKey
	}
	if f.APIProvider != nil {
		cfg["api-provider"] = *f.APIProvider
	}
	if f.APIEndpoint != nil {
		cfg["api-endpoint"] = *f.APIEndpoint
	}
	if f.DefaultMode != nil {
		cfg["default-mode"] = *f.DefaultMode
	}
	if f.WorkspaceDir != nil {
		cfg["workspace-dir"] = *f.WorkspaceDir
	}
	if f.MaxConcurrent != nil {
		cfg["max-concurrent"] = *f.MaxConcurrent
	}
	if f.EnableSandbox != nil {
		cfg["enable-sandbox"] = *f.EnableSandbox
	}
	if f.AgentPort != nil {
		cfg["agent-port"] = *f.AgentPort
	}
	if f.LogLevel != nil {
		cfg["log-level"] = *f.LogLevel
	}
	if f.LogFile != nil {
		cfg["log-file"] = *f.LogFile
	}

	return cfg
}


