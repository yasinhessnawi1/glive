package prompts

import "fmt"

// JSONSchema represents a JSON schema definition
type JSONSchema struct {
	Name       string                    `json:"name"`
	Type       string                    `json:"type"`
	Required   []string                  `json:"required,omitempty"`
	Properties map[string]SchemaProperty `json:"properties,omitempty"`
	Items      *JSONSchema               `json:"items,omitempty"`
	Enum       []interface{}             `json:"enum,omitempty"`
	MinLength  *int                      `json:"minLength,omitempty"`
	MaxLength  *int                      `json:"maxLength,omitempty"`
	Minimum    *float64                  `json:"minimum,omitempty"`
	Maximum    *float64                  `json:"maximum,omitempty"`
	Pattern    string                    `json:"pattern,omitempty"`
	MinItems   *int                      `json:"minItems,omitempty"`
	MaxItems   *int                      `json:"maxItems,omitempty"`
}

// SchemaProperty represents a property in a JSON schema
type SchemaProperty struct {
	Type        string                    `json:"type"`
	Description string                    `json:"description,omitempty"`
	Enum        []interface{}             `json:"enum,omitempty"`
	MinLength   *int                      `json:"minLength,omitempty"`
	MaxLength   *int                      `json:"maxLength,omitempty"`
	Minimum     *float64                  `json:"minimum,omitempty"`
	Maximum     *float64                  `json:"maximum,omitempty"`
	Pattern     string                    `json:"pattern,omitempty"`
	Items       *JSONSchema               `json:"items,omitempty"`
	Properties  map[string]SchemaProperty `json:"properties,omitempty"`
	Required    []string                  `json:"required,omitempty"`
	MinItems    *int                      `json:"minItems,omitempty"`
	MaxItems    *int                      `json:"maxItems,omitempty"`
}

// RecoveryPlanSchema returns the JSON schema for error recovery plans
func RecoveryPlanSchema() JSONSchema {
	minLength10 := 10
	maxLength500 := 500
	maxLength1000 := 1000
	minItems1 := 1
	maxItems10 := 10
	minConfidence := 0.0
	maxConfidence := 1.0

	return JSONSchema{
		Name:     "RecoveryPlanSchema",
		Type:     "object",
		Required: []string{"error_type", "root_cause", "confidence", "recovery_steps"},
		Properties: map[string]SchemaProperty{
			"error_type": {
				Type:        "string",
				Description: "Type of error encountered",
				Enum: []interface{}{
					"missing_dependency",
					"config_error",
					"build_failure",
					"permission_error",
					"network_error",
					"environment_error",
					"version_conflict",
					"syntax_error",
					"runtime_error",
					"unknown_error",
				},
			},
			"root_cause": {
				Type:        "string",
				Description: "Root cause analysis of the error",
				MinLength:   &minLength10,
				MaxLength:   &maxLength500,
			},
			"confidence": {
				Type:        "number",
				Description: "Confidence level in the diagnosis (0.0-1.0)",
				Minimum:     &minConfidence,
				Maximum:     &maxConfidence,
			},
			"recovery_steps": {
				Type:        "array",
				Description: "Steps to recover from the error",
				MinItems:    &minItems1,
				MaxItems:    &maxItems10,
				Items: &JSONSchema{
					Type:     "object",
					Required: []string{"command", "reason", "risk_level"},
					Properties: map[string]SchemaProperty{
						"command": {
							Type:        "string",
							Description: "Command to execute",
							Pattern:     "^[a-zA-Z0-9\\s\\-\\./_]+$",
						},
						"reason": {
							Type:        "string",
							Description: "Reason for this recovery step",
						},
						"risk_level": {
							Type:        "string",
							Description: "Risk level of this step",
							Enum: []interface{}{
								"low",
								"medium",
								"high",
								"critical",
							},
						},
						"can_rollback": {
							Type:        "boolean",
							Description: "Whether this step can be rolled back",
						},
						"rollback_command": {
							Type:        "string",
							Description: "Command to rollback this step",
						},
					},
				},
			},
			"manual_steps_if_failed": {
				Type:        "array",
				Description: "Manual steps if automatic recovery fails",
				Items: &JSONSchema{
					Type: "string",
				},
			},
			"explanation": {
				Type:        "string",
				Description: "Explanation for the user",
				MaxLength:   &maxLength1000,
			},
		},
	}
}

// ProjectAnalysisSchema returns the JSON schema for project analysis
func ProjectAnalysisSchema() JSONSchema {
	maxItems50 := 50

	return JSONSchema{
		Name:     "ProjectAnalysisSchema",
		Type:     "object",
		Required: []string{"project_type", "commands"},
		Properties: map[string]SchemaProperty{
			"project_type": {
				Type:        "string",
				Description: "Type of project",
				Enum: []interface{}{
					"nodejs",
					"python",
					"go",
					"rust",
					"java",
					"docker",
					"polyglot",
					"unknown",
				},
			},
			"detected_languages": {
				Type:        "array",
				Description: "Detected programming languages",
				Items: &JSONSchema{
					Type: "string",
				},
			},
			"package_managers": {
				Type:        "array",
				Description: "Package managers detected",
				Items: &JSONSchema{
					Type: "string",
				},
			},
			"entry_points": {
				Type:        "array",
				Description: "Entry point files",
				Items: &JSONSchema{
					Type: "string",
				},
			},
			"dependencies": {
				Type:        "array",
				Description: "Project dependencies",
				Items: &JSONSchema{
					Type: "object",
					Properties: map[string]SchemaProperty{
						"name": {
							Type: "string",
						},
						"version": {
							Type: "string",
						},
						"type": {
							Type: "string",
						},
						"installed": {
							Type: "boolean",
						},
					},
				},
			},
			"system_requirements": {
				Type:        "array",
				Description: "System requirements",
				Items: &JSONSchema{
					Type: "string",
				},
			},
			"commands": {
				Type:        "array",
				Description: "Setup commands",
				MaxItems:    &maxItems50,
				Items: &JSONSchema{
					Type:     "object",
					Required: []string{"command", "working_dir", "stage"},
					Properties: map[string]SchemaProperty{
						"id": {
							Type: "string",
						},
						"description": {
							Type: "string",
						},
						"command": {
							Type:        "string",
							Description: "Command to execute",
						},
						"working_dir": {
							Type:        "string",
							Description: "Working directory",
						},
						"stage": {
							Type:        "string",
							Description: "Command stage",
							Enum: []interface{}{
								"setup",
								"build",
								"run",
								"test",
							},
						},
						"required": {
							Type:        "boolean",
							Description: "Whether command is required",
						},
						"status": {
							Type:        "string",
							Description: "Command status",
							Enum: []interface{}{
								"pending",
								"running",
								"completed",
								"failed",
							},
						},
					},
				},
			},
			"is_suspicious": {
				Type:        "boolean",
				Description: "Whether project is suspicious",
			},
			"suspicious_reasons": {
				Type:        "array",
				Description: "Reasons if suspicious",
				Items: &JSONSchema{
					Type: "string",
				},
			},
			"estimated_size": {
				Type:        "string",
				Description: "Estimated project size",
			},
			"description": {
				Type:        "string",
				Description: "Project description",
			},
		},
	}
}

// SecurityScanSchema returns the JSON schema for security scans
func SecurityScanSchema() JSONSchema {
	return JSONSchema{
		Name:     "SecurityScanSchema",
		Type:     "object",
		Required: []string{"is_safe", "threats"},
		Properties: map[string]SchemaProperty{
			"is_safe": {
				Type:        "boolean",
				Description: "Whether project is safe",
			},
			"threats": {
				Type:        "array",
				Description: "Detected security threats",
				Items: &JSONSchema{
					Type:     "object",
					Required: []string{"type", "severity", "description"},
					Properties: map[string]SchemaProperty{
						"type": {
							Type:        "string",
							Description: "Threat type",
							Enum: []interface{}{
								"malware",
								"credential_leak",
								"vulnerability",
								"suspicious_command",
								"other",
							},
						},
						"severity": {
							Type:        "string",
							Description: "Threat severity",
							Enum: []interface{}{
								"low",
								"medium",
								"high",
								"critical",
							},
						},
						"description": {
							Type:        "string",
							Description: "Threat description",
						},
						"file_path": {
							Type:        "string",
							Description: "File path where threat was found",
						},
						"line_number": {
							Type:        "number",
							Description: "Line number where threat was found",
						},
					},
				},
			},
			"confidence": {
				Type:        "number",
				Description: "Confidence in scan results (0.0-1.0)",
				Minimum:     floatPtr(0.0),
				Maximum:     floatPtr(1.0),
			},
		},
	}
}

// CommandGenerationSchema returns the JSON schema for command generation
func CommandGenerationSchema() JSONSchema {
	maxItems20 := 20

	return JSONSchema{
		Name:     "CommandGenerationSchema",
		Type:     "object",
		Required: []string{"commands"},
		Properties: map[string]SchemaProperty{
			"commands": {
				Type:        "array",
				Description: "Generated commands",
				MaxItems:    &maxItems20,
				Items: &JSONSchema{
					Type:     "object",
					Required: []string{"command", "description", "order"},
					Properties: map[string]SchemaProperty{
						"command": {
							Type:        "string",
							Description: "Command to execute",
						},
						"description": {
							Type:        "string",
							Description: "Command description",
						},
						"order": {
							Type:        "number",
							Description: "Execution order",
						},
						"working_dir": {
							Type:        "string",
							Description: "Working directory",
						},
						"dependencies": {
							Type:        "array",
							Description: "Command dependencies",
							Items: &JSONSchema{
								Type: "number",
							},
						},
					},
				},
			},
			"estimated_time": {
				Type:        "number",
				Description: "Estimated execution time in seconds",
			},
		},
	}
}

// AutoFixSchema returns the JSON schema for auto-fix responses
func AutoFixSchema() JSONSchema {
	return JSONSchema{
		Name:     "AutoFixSchema",
		Type:     "object",
		Required: []string{"can_auto_fix", "explanation", "confidence"},
		Properties: map[string]SchemaProperty{
			"can_auto_fix": {
				Type:        "boolean",
				Description: "Whether the error can be auto-fixed",
			},
			"fixed_command": {
				Type:        "string",
				Description: "Fixed command (only if can_auto_fix is true)",
			},
			"explanation": {
				Type:        "string",
				Description: "Explanation of the fix",
			},
			"confidence": {
				Type:        "string",
				Description: "Confidence level",
				Enum: []interface{}{
					"high",
					"medium",
					"low",
				},
			},
		},
	}
}

// GetSchemaByName returns a schema by name
func GetSchemaByName(name string) (JSONSchema, error) {
	switch name {
	case "RecoveryPlan":
		return RecoveryPlanSchema(), nil
	case "ProjectAnalysis":
		return ProjectAnalysisSchema(), nil
	case "SecurityScan":
		return SecurityScanSchema(), nil
	case "CommandGeneration":
		return CommandGenerationSchema(), nil
	case "AutoFix":
		return AutoFixSchema(), nil
	default:
		return JSONSchema{}, fmt.Errorf("unknown schema: %s", name)
	}
}

// floatPtr returns a pointer to a float64
func floatPtr(f float64) *float64 {
	return &f
}
