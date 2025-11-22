package prompts

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// SchemaValidator validates JSON responses against schemas
type SchemaValidator struct {
	schemas map[string]JSONSchema
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() *SchemaValidator {
	sv := &SchemaValidator{
		schemas: make(map[string]JSONSchema),
	}

	// Register default schemas
	sv.schemas["RecoveryPlan"] = RecoveryPlanSchema()
	sv.schemas["ProjectAnalysis"] = ProjectAnalysisSchema()
	sv.schemas["SecurityScan"] = SecurityScanSchema()
	sv.schemas["CommandGeneration"] = CommandGenerationSchema()
	sv.schemas["AutoFix"] = AutoFixSchema()

	return sv
}

// Validate validates a JSON response against a schema
func (sv *SchemaValidator) Validate(schemaName string, response interface{}) error {
	schema, ok := sv.schemas[schemaName]
	if !ok {
		return fmt.Errorf("schema %s not found", schemaName)
	}

	// Convert response to map[string]interface{} if needed
	var data map[string]interface{}
	switch v := response.(type) {
	case map[string]interface{}:
		data = v
	case string:
		if err := json.Unmarshal([]byte(v), &data); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
	case []byte:
		if err := json.Unmarshal(v, &data); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
	default:
		// Try to convert via JSON
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal response: %w", err)
		}
		if err := json.Unmarshal(jsonBytes, &data); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
	}

	return sv.validateObject(data, schema)
}

// validateObject validates an object against a schema
func (sv *SchemaValidator) validateObject(obj map[string]interface{}, schema JSONSchema) error {
	// Check required fields
	for _, field := range schema.Required {
		if _, ok := obj[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Validate each property
	for propName, propSchema := range schema.Properties {
		value, exists := obj[propName]
		if !exists {
			continue // Optional field
		}

		if err := sv.validateProperty(value, propSchema); err != nil {
			return fmt.Errorf("field '%s': %w", propName, err)
		}
	}

	return nil
}

// validateProperty validates a property value against its schema
func (sv *SchemaValidator) validateProperty(value interface{}, prop SchemaProperty) error {
	// Type validation
	if err := sv.validateType(value, prop.Type); err != nil {
		return err
	}

	// String validations
	if prop.Type == "string" {
		strValue, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", value)
		}

		if prop.MinLength != nil && len(strValue) < *prop.MinLength {
			return fmt.Errorf("string length %d is less than minimum %d", len(strValue), *prop.MinLength)
		}

		if prop.MaxLength != nil && len(strValue) > *prop.MaxLength {
			return fmt.Errorf("string length %d exceeds maximum %d", len(strValue), *prop.MaxLength)
		}

		if prop.Pattern != "" {
			matched, _ := regexp.MatchString(prop.Pattern, strValue)
			if !matched {
				return fmt.Errorf("string does not match pattern: %s", prop.Pattern)
			}
		}

		if len(prop.Enum) > 0 {
			found := false
			for _, enumVal := range prop.Enum {
				if enumVal == strValue {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("value '%s' is not in enum: %v", strValue, prop.Enum)
			}
		}
	}

	// Number validations
	if prop.Type == "number" {
		numValue, err := sv.toFloat64(value)
		if err != nil {
			return fmt.Errorf("expected number, got %T", value)
		}

		if prop.Minimum != nil && numValue < *prop.Minimum {
			return fmt.Errorf("number %f is less than minimum %f", numValue, *prop.Minimum)
		}

		if prop.Maximum != nil && numValue > *prop.Maximum {
			return fmt.Errorf("number %f exceeds maximum %f", numValue, *prop.Maximum)
		}
	}

	// Boolean validation
	if prop.Type == "boolean" {
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	}

	// Array validations
	if prop.Type == "array" {
		arrValue, ok := value.([]interface{})
		if !ok {
			return fmt.Errorf("expected array, got %T", value)
		}

		if prop.MinItems != nil && len(arrValue) < *prop.MinItems {
			return fmt.Errorf("array length %d is less than minimum %d", len(arrValue), *prop.MinItems)
		}

		if prop.MaxItems != nil && len(arrValue) > *prop.MaxItems {
			return fmt.Errorf("array length %d exceeds maximum %d", len(arrValue), *prop.MaxItems)
		}

		if prop.Items != nil {
			for i, item := range arrValue {
				if err := sv.validateProperty(item, SchemaProperty{
					Type: prop.Items.Type,
					Properties: prop.Items.Properties,
					Required: prop.Items.Required,
				}); err != nil {
					return fmt.Errorf("array item %d: %w", i, err)
				}
			}
		}
	}

	// Object validations
	if prop.Type == "object" {
		objValue, ok := value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected object, got %T", value)
		}

		if len(prop.Properties) > 0 || len(prop.Required) > 0 {
			objSchema := JSONSchema{
				Type:       "object",
				Required:   prop.Required,
				Properties: prop.Properties,
			}
			if err := sv.validateObject(objValue, objSchema); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateType validates the type of a value
func (sv *SchemaValidator) validateType(value interface{}, expectedType string) error {
	if value == nil {
		return fmt.Errorf("value is nil")
	}

	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "number":
		if _, err := sv.toFloat64(value); err != nil {
			return fmt.Errorf("expected number, got %T", value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("expected array, got %T", value)
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("expected object, got %T", value)
		}
	}

	return nil
}

// toFloat64 converts a value to float64
func (sv *SchemaValidator) toFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, err
		}
		return f, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

// ExtractJSONFromResponse extracts JSON from a response string that may contain markdown
func ExtractJSONFromResponse(response string) (string, error) {
	// Try to find JSON in markdown code blocks (```json)
	codeBlockStart := strings.Index(response, "```json")
	if codeBlockStart >= 0 {
		contentStart := codeBlockStart + 7 // Skip "```json"
		// Skip to next line
		newlinePos := strings.Index(response[contentStart:], "\n")
		if newlinePos >= 0 {
			contentStart += newlinePos + 1
		}
		// Find closing ```
		codeBlockEnd := strings.Index(response[contentStart:], "```")
		if codeBlockEnd >= 0 {
			jsonStr := strings.TrimSpace(response[contentStart : contentStart+codeBlockEnd])
			var test interface{}
			if json.Unmarshal([]byte(jsonStr), &test) == nil {
				return jsonStr, nil
			}
		}
	}

	// Try to find JSON in regular code blocks (```)
	codeBlockStart = strings.Index(response, "```")
	if codeBlockStart >= 0 {
		contentStart := codeBlockStart + 3 // Skip "```"
		// Skip to next line
		newlinePos := strings.Index(response[contentStart:], "\n")
		if newlinePos >= 0 {
			contentStart += newlinePos + 1
		}
		// Find closing ```
		codeBlockEnd := strings.Index(response[contentStart:], "```")
		if codeBlockEnd >= 0 {
			jsonStr := strings.TrimSpace(response[contentStart : contentStart+codeBlockEnd])
			// Validate it's JSON
			var test interface{}
			if json.Unmarshal([]byte(jsonStr), &test) == nil {
				return jsonStr, nil
			}
		}
	}

	// Try to find JSON object directly
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start >= 0 && end > start {
		jsonStr := response[start : end+1]
		// Validate it's JSON
		var test interface{}
		if json.Unmarshal([]byte(jsonStr), &test) == nil {
			return jsonStr, nil
		}
	}

	return "", fmt.Errorf("no valid JSON found in response")
}

