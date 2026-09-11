package validation

import (
	"encoding/json"
	"fmt"

	"github.com/glive/domain/errors"
	"github.com/glive/domain/values"
)

// JSONValidator validates JSON responses
type JSONValidator struct {
	maxSize      int64
	maxDepth     int
	maxArrayLen  int
	maxStringLen int
}

// NewJSONValidator creates a new JSON validator
func NewJSONValidator() *JSONValidator {
	return &JSONValidator{
		maxSize:      10 * 1024 * 1024, // 10MB
		maxDepth:     20,
		maxArrayLen:  1000,
		maxStringLen: 100000,
	}
}

// Validate validates JSON data
func (v *JSONValidator) Validate(data []byte) error {
	// Check size
	if int64(len(data)) > v.maxSize {
		return errors.NewUserError("JSON_TOO_LARGE",
			fmt.Sprintf("JSON exceeds maximum size of %d bytes", v.maxSize))
	}

	// Try to parse
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return errors.NewUserError("JSON_INVALID",
			fmt.Sprintf("Invalid JSON: %v", err))
	}

	// Validate structure
	return v.validateValue(raw, 0)
}

func (v *JSONValidator) validateValue(val interface{}, depth int) error {
	if depth > v.maxDepth {
		return errors.NewSecurityError("JSON_TOO_DEEP",
			fmt.Sprintf("JSON exceeds maximum depth of %d", v.maxDepth))
	}

	switch typed := val.(type) {
	case map[string]interface{}:
		for key, value := range typed {
			if len(key) > v.maxStringLen {
				return errors.NewSecurityError("JSON_KEY_TOO_LONG",
					"JSON object key exceeds maximum length")
			}
			if err := v.validateValue(value, depth+1); err != nil {
				return err
			}
		}

	case []interface{}:
		if len(typed) > v.maxArrayLen {
			return errors.NewSecurityError("JSON_ARRAY_TOO_LONG",
				fmt.Sprintf("JSON array exceeds maximum length of %d", v.maxArrayLen))
		}
		for _, item := range typed {
			if err := v.validateValue(item, depth+1); err != nil {
				return err
			}
		}

	case string:
		if len(typed) > v.maxStringLen {
			return errors.NewSecurityError("JSON_STRING_TOO_LONG",
				fmt.Sprintf("JSON string exceeds maximum length of %d", v.maxStringLen))
		}

	case float64, bool, nil:
		// These are fine

	default:
		return errors.NewUserError("JSON_UNKNOWN_TYPE",
			fmt.Sprintf("Unknown JSON type: %T", val))
	}

	return nil
}

// ValidatedAnalysis represents a validated AI analysis response
type ValidatedAnalysis struct {
	ProjectType       string
	Commands          []*values.ValidatedCommand
	IsSuspicious      bool
	SuspiciousReasons []string
}

// AIResponseValidator validates AI API responses
type AIResponseValidator struct {
	jsonValidator    *JSONValidator
	commandValidator *values.CommandValidator
	maxCommands      int
}

// NewAIResponseValidator creates a new AI response validator
func NewAIResponseValidator() *AIResponseValidator {
	return &AIResponseValidator{
		jsonValidator:    NewJSONValidator(),
		commandValidator: values.NewCommandValidator(),
		maxCommands:      100,
	}
}

// ValidateAnalysis validates an AI analysis response
func (v *AIResponseValidator) ValidateAnalysis(response []byte, workingDir string) (*ValidatedAnalysis, error) {
	// Validate JSON structure
	if err := v.jsonValidator.Validate(response); err != nil {
		return nil, err
	}

	// Parse response
	var raw struct {
		ProjectType       string                   `json:"project_type"`
		Commands          []map[string]interface{} `json:"commands"`
		IsSuspicious      bool                     `json:"is_suspicious"`
		SuspiciousReasons []string                 `json:"suspicious_reasons"`
	}

	if err := json.Unmarshal(response, &raw); err != nil {
		return nil, errors.WrapWithCode(err, "AI_PARSE_ERROR", "failed to parse AI response")
	}

	// Validate command count
	if len(raw.Commands) > v.maxCommands {
		return nil, errors.NewSecurityError("AI_TOO_MANY_COMMANDS",
			fmt.Sprintf("AI response contains too many commands (%d > %d)",
				len(raw.Commands), v.maxCommands))
	}

	// Validate each command
	validatedCommands := make([]*values.ValidatedCommand, 0, len(raw.Commands))
	for i, cmdRaw := range raw.Commands {
		cmdStr, ok := cmdRaw["command"].(string)
		if !ok {
			return nil, errors.NewUserError("AI_INVALID_COMMAND",
				fmt.Sprintf("Command %d: missing command string", i))
		}

		cmdWorkDir := workingDir
		if wd, ok := cmdRaw["working_dir"].(string); ok && wd != "" {
			cmdWorkDir = wd
		}

		validated, err := v.commandValidator.Validate(cmdStr, cmdWorkDir)
		if err != nil {
			return nil, errors.WrapWithCode(err, "AI_CMD_VALIDATION_FAILED",
				fmt.Sprintf("command %d validation failed", i))
		}

		validatedCommands = append(validatedCommands, validated)
	}

	return &ValidatedAnalysis{
		ProjectType:       raw.ProjectType,
		Commands:          validatedCommands,
		IsSuspicious:      raw.IsSuspicious,
		SuspiciousReasons: raw.SuspiciousReasons,
	}, nil
}
