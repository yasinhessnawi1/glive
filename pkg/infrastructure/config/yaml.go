package config

import (
	"gopkg.in/yaml.v3"
)

// parseYAML parses YAML data into a configuration map
func parseYAML(data []byte) (map[string]interface{}, error) {
	var cfg map[string]interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
