package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/glive/infrastructure/security"
	"github.com/glive/infrastructure/validation"
)

// ParsedCommand represents a command parsed from AI response
// This is a local type to avoid import cycles with executor package
type ParsedCommand struct {
	ID          string `json:"id"`
	Command     string `json:"command"`
	Description string `json:"description"`
	WorkingDir  string `json:"working_dir"`
	Stage       string `json:"stage"`
	Required    bool   `json:"required"`
	Order       int    `json:"order"`
}

// ResponseParser parses and validates AI responses
type ResponseParser struct {
	validator *validation.JSONValidator
	sanitizer *security.CommandValidator
}

// NewResponseParser creates a new response parser
func NewResponseParser() *ResponseParser {
	return &ResponseParser{
		validator: validation.NewJSONValidator(),
		sanitizer: security.NewCommandValidator(),
	}
}

// Analysis represents a parsed analysis response
type Analysis struct {
	ProjectType        string                   `json:"project_type"`
	DetectedLanguages  []string                 `json:"detected_languages"`
	PackageManagers    []string                 `json:"package_managers"`
	EntryPoints        []string                 `json:"entry_points"`
	Dependencies       []map[string]interface{} `json:"dependencies"`
	SystemRequirements []string                 `json:"system_requirements"`
	Commands           []*ParsedCommand          `json:"commands"`
	IsSuspicious       bool                     `json:"is_suspicious"`
	SuspiciousReasons  []string                 `json:"suspicious_reasons"`
	EstimatedSize      string                   `json:"estimated_size"`
	Description        string                   `json:"description"`
}

// ParseAnalysis parses an AI analysis response
func (p *ResponseParser) ParseAnalysis(response string) (*Analysis, error) {
	// 1. Extract JSON from response (handle markdown code blocks)
	jsonStr := extractJSON(response)

	// 2. Validate JSON structure
	if err := p.validator.Validate([]byte(jsonStr)); err != nil {
		return nil, fmt.Errorf("JSON validation failed: %w", err)
	}

	// 3. Parse into map first to handle dynamic structure
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// 4. Build Analysis struct
	analysis := &Analysis{}

	// Extract fields safely
	if pt, ok := raw["project_type"].(string); ok {
		analysis.ProjectType = pt
	}
	if dl, ok := raw["detected_languages"].([]interface{}); ok {
		analysis.DetectedLanguages = toStringSlice(dl)
	}
	if pm, ok := raw["package_managers"].([]interface{}); ok {
		analysis.PackageManagers = toStringSlice(pm)
	}
	if ep, ok := raw["entry_points"].([]interface{}); ok {
		analysis.EntryPoints = toStringSlice(ep)
	}
	if deps, ok := raw["dependencies"].([]interface{}); ok {
		analysis.Dependencies = make([]map[string]interface{}, 0, len(deps))
		for _, dep := range deps {
			if depMap, ok := dep.(map[string]interface{}); ok {
				analysis.Dependencies = append(analysis.Dependencies, depMap)
			}
		}
	}
	if sr, ok := raw["system_requirements"].([]interface{}); ok {
		analysis.SystemRequirements = toStringSlice(sr)
	}
	if sus, ok := raw["is_suspicious"].(bool); ok {
		analysis.IsSuspicious = sus
	}
	if srs, ok := raw["suspicious_reasons"].([]interface{}); ok {
		analysis.SuspiciousReasons = toStringSlice(srs)
	}
	if size, ok := raw["estimated_size"].(string); ok {
		analysis.EstimatedSize = size
	}
	if desc, ok := raw["description"].(string); ok {
		analysis.Description = desc
	}

	// 5. Parse and validate commands
	if commandsRaw, ok := raw["commands"].([]interface{}); ok {
		analysis.Commands = make([]*ParsedCommand, 0, len(commandsRaw))
		for i, cmdRaw := range commandsRaw {
			cmdMap, ok := cmdRaw.(map[string]interface{})
			if !ok {
				continue
			}

			cmdStr, ok := cmdMap["command"].(string)
			if !ok || cmdStr == "" {
				continue
			}

			// Validate command security
			if err := p.sanitizer.Validate(cmdStr); err != nil {
				return nil, fmt.Errorf("command %d validation failed: %w", i, err)
			}

			// Build command object
			cmd := &ParsedCommand{
				Command: cmdStr,
			}

			if id, ok := cmdMap["id"].(string); ok {
				cmd.ID = id
			} else {
				cmd.ID = fmt.Sprintf("cmd-%d", i)
			}
			if desc, ok := cmdMap["description"].(string); ok {
				cmd.Description = desc
			}
			if wd, ok := cmdMap["working_dir"].(string); ok {
				cmd.WorkingDir = wd
			}
			if stage, ok := cmdMap["stage"].(string); ok {
				cmd.Stage = stage
			}
			if req, ok := cmdMap["required"].(bool); ok {
				cmd.Required = req
			} else {
				cmd.Required = true
			}

			analysis.Commands = append(analysis.Commands, cmd)
		}
	}

	return analysis, nil
}

// extractJSON extracts JSON from a response string, handling markdown code blocks
func extractJSON(response string) string {
	// Try to find JSON in markdown code blocks first
	codeBlockRegex := regexp.MustCompile("(?s)```(?:json)?\\s*\\{.*\\}\\s*```")
	if matches := codeBlockRegex.FindString(response); matches != "" {
		// Extract JSON from code block
		jsonStart := strings.Index(matches, "{")
		jsonEnd := strings.LastIndex(matches, "}")
		if jsonStart >= 0 && jsonEnd > jsonStart {
			return matches[jsonStart : jsonEnd+1]
		}
	}

	// Try to find JSON object directly
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		return response[jsonStart : jsonEnd+1]
	}

	// If no JSON found, return original (will fail validation)
	return response
}

