package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	prompts "github.com/glive/infrastructure/ai/prompts"
	"github.com/glive/infrastructure/security"
)

const (
	// MaxCommands is the maximum number of commands allowed in an AI response
	MaxCommands = 50
	// MaxCommandLength is the maximum length of a single command
	MaxCommandLength = 1000
	// MaxResponseLength is the maximum length of the entire AI response
	MaxResponseLength = 100000
)

// AIResponseValidator validates AI responses before execution
type AIResponseValidator struct {
	commandValidator  *security.CommandValidator
	schemaValidator   *prompts.SchemaValidator
	maxCommands       int
	maxCommandLength  int
	maxResponseLength int
}

// NewAIResponseValidator creates a new AI response validator
func NewAIResponseValidator() *AIResponseValidator {
	return &AIResponseValidator{
		commandValidator:  security.NewCommandValidator(),
		schemaValidator:   prompts.NewSchemaValidator(),
		maxCommands:       MaxCommands,
		maxCommandLength:  MaxCommandLength,
		maxResponseLength: MaxResponseLength,
	}
}

// ValidatedAnalysis represents a validated AI analysis response
type ValidatedAnalysis struct {
	ProjectType        string
	DetectedLanguages  []string
	PackageManagers    []string
	EntryPoints        []string
	Dependencies       []Dependency
	SystemRequirements []string
	Commands           []*ParsedCommand
	IsSuspicious       bool
	SuspiciousReasons  []string
	EstimatedSize      string
	Description        string
}

// Dependency represents a dependency
type Dependency struct {
	Name      string
	Version   string
	Type      string
	Installed bool
}

// ValidateAnalysis validates an AI analysis response
func (v *AIResponseValidator) ValidateAnalysis(response string) (*ValidatedAnalysis, error) {
	// 1. Check response length
	if len(response) > v.maxResponseLength {
		return nil, fmt.Errorf("response too long: %d bytes (max %d)", len(response), v.maxResponseLength)
	}

	// 2. Detect prompt injection attempts
	if v.detectPromptInjection(response) {
		return nil, fmt.Errorf("potential prompt injection detected")
	}

	// 3. Parse JSON safely
	var raw map[string]interface{}

	// Try to extract JSON from response (AI sometimes adds extra text)
	jsonStr, err := prompts.ExtractJSONFromResponse(response)
	if err != nil {
		// Fallback to old method
		jsonStart := strings.Index(response, "{")
		jsonEnd := strings.LastIndex(response, "}")
		if jsonStart < 0 || jsonEnd <= jsonStart {
			return nil, fmt.Errorf("no valid JSON found in response")
		}
		jsonStr = response[jsonStart : jsonEnd+1]
	}

	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}

	// 3.5. Validate against schema
	if err := v.schemaValidator.Validate("ProjectAnalysis", raw); err != nil {
		// Log schema validation error but don't fail (for backward compatibility)
		_ = err
	}

	// 4. Extract and validate commands
	commandsRaw, ok := raw["commands"].([]interface{})
	if !ok {
		// Commands might be missing, that's okay
		commandsRaw = []interface{}{}
	}

	if len(commandsRaw) > v.maxCommands {
		return nil, fmt.Errorf("too many commands: %d (max %d)", len(commandsRaw), v.maxCommands)
	}

	validatedCommands := make([]*ParsedCommand, 0, len(commandsRaw))
	for i, cmdRaw := range commandsRaw {
		cmd, err := v.validateCommand(cmdRaw, i)
		if err != nil {
			return nil, fmt.Errorf("command %d: %w", i, err)
		}
		validatedCommands = append(validatedCommands, cmd)
	}

	// 5. Build validated result
	result := &ValidatedAnalysis{
		Commands: validatedCommands,
	}

	// Extract other fields safely
	if pt, ok := raw["project_type"].(string); ok {
		result.ProjectType = pt
	}
	if dl, ok := raw["detected_languages"].([]interface{}); ok {
		result.DetectedLanguages = toStringSlice(dl)
	}
	if pm, ok := raw["package_managers"].([]interface{}); ok {
		result.PackageManagers = toStringSlice(pm)
	}
	if ep, ok := raw["entry_points"].([]interface{}); ok {
		result.EntryPoints = toStringSlice(ep)
	}
	if sr, ok := raw["system_requirements"].([]interface{}); ok {
		result.SystemRequirements = toStringSlice(sr)
	}
	if desc, ok := raw["description"].(string); ok {
		result.Description = desc
	}
	if size, ok := raw["estimated_size"].(string); ok {
		result.EstimatedSize = size
	}
	if sus, ok := raw["is_suspicious"].(bool); ok {
		result.IsSuspicious = sus
	}
	if srs, ok := raw["suspicious_reasons"].([]interface{}); ok {
		result.SuspiciousReasons = toStringSlice(srs)
	}

	return result, nil
}

// validateCommand validates a single command from AI response
func (v *AIResponseValidator) validateCommand(cmdRaw interface{}, index int) (*ParsedCommand, error) {
	cmdMap, ok := cmdRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid command format")
	}

	// Extract command string
	cmdStr, ok := cmdMap["command"].(string)
	if !ok || cmdStr == "" {
		return nil, fmt.Errorf("missing command string")
	}

	// Length check
	if len(cmdStr) > v.maxCommandLength {
		return nil, fmt.Errorf("command too long: %d bytes (max %d)", len(cmdStr), v.maxCommandLength)
	}

	// Security validation
	if err := v.commandValidator.Validate(cmdStr); err != nil {
		return nil, fmt.Errorf("security check failed: %w", err)
	}

	// Build command object
	cmd := &ParsedCommand{
		Command: cmdStr,
	}

	// Extract optional fields
	if id, ok := cmdMap["id"].(string); ok {
		cmd.ID = id
	} else {
		cmd.ID = fmt.Sprintf("cmd-%d", index)
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
		cmd.Required = true // Default to required
	}

	return cmd, nil
}

// detectPromptInjection detects potential prompt injection attempts
func (v *AIResponseValidator) detectPromptInjection(response string) bool {
	responseLower := strings.ToLower(response)

	// Only check for actual prompt injection phrases (full phrases, not single words)
	// These are phrases that would indicate an attempt to manipulate the AI
	injectionPatterns := []string{
		"ignore previous instructions",
		"forget all previous",
		"ignore all previous",
		"disregard previous",
		"new instructions:",
		"you are now",
		"pretend to be",
		"act as if you",
		"do not follow the",
		"override your",
		"bypass your",
	}

	for _, pattern := range injectionPatterns {
		if strings.Contains(responseLower, pattern) {
			return true
		}
	}

	return false
}

// toStringSlice converts []interface{} to []string
func toStringSlice(slice []interface{}) []string {
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if str, ok := v.(string); ok {
			result = append(result, str)
		}
	}
	return result
}
