package config

import (
	"encoding/json"
)

// parseJSON parses JSON data into a configuration map
func parseJSON(data []byte) (map[string]interface{}, error) {
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
