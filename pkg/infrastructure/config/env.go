package config

import (
	"os"
	"strconv"
	"strings"
)

// applyEnv applies environment variables to the configuration
func (l *ConfigLoader) applyEnv(cfg map[string]interface{}) map[string]interface{} {
	mappings := map[string]string{
		"GLIVE_API_KEY":        "api-key",
		"GLIVE_API_PROVIDER":   "api-provider",
		"GLIVE_API_ENDPOINT":   "api-endpoint",
		"GLIVE_MODE":           "default-mode",
		"GLIVE_WORKSPACE_DIR":  "workspace-dir",
		"GLIVE_SANDBOX":        "enable-sandbox",
		"GLIVE_AGENT_PORT":     "agent-port",
		"GLIVE_MAX_CONCURRENT": "max-concurrent",
		"GLIVE_LOG_LEVEL":      "log-level",
		"GLIVE_LOG_FILE":       "log-file",
	}

	for env, key := range mappings {
		if val := os.Getenv(env); val != "" {
			// Convert value based on expected type
			convertedVal := convertEnvValue(key, val)
			cfg[key] = convertedVal
		}
	}

	return cfg
}

// convertEnvValue converts environment variable string to appropriate type
func convertEnvValue(key, value string) interface{} {
	// Boolean keys
	if strings.HasPrefix(key, "enable-") || key == "enable-sandbox" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
		return value == "true" || value == "1" || value == "yes"
	}

	// Integer keys
	intKeys := []string{"agent-port", "max-concurrent"}
	for _, intKey := range intKeys {
		if key == intKey {
			if i, err := strconv.Atoi(value); err == nil {
				return i
			}
			return 0
		}
	}

	// String keys (default)
	return value
}
