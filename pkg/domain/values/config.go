package values

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/glive/domain/errors"
)

// Validator interface for custom validation
type Validator interface {
	Validate() error
}

// ValidationRule defines a validation rule
type ValidationRule struct {
	Field    string
	Required bool
	Type     string
	Min      interface{}
	Max      interface{}
	Pattern  *regexp.Regexp
	OneOf    []interface{}
	Custom   func(interface{}) error
}

// Schema defines validation schema for a config
type Schema struct {
	rules map[string]ValidationRule
}

// NewSchema creates a new validation schema
func NewSchema() *Schema {
	return &Schema{rules: make(map[string]ValidationRule)}
}

// AddRule adds a validation rule to the schema
func (s *Schema) AddRule(rule ValidationRule) *Schema {
	s.rules[rule.Field] = rule
	return s
}

// Validate validates data against the schema
func (s *Schema) Validate(data map[string]interface{}) []ValidationError {
	var errs []ValidationError

	for field, rule := range s.rules {
		value, exists := data[field]

		// Check required
		if rule.Required && !exists {
			errs = append(errs, ValidationError{
				Field:   field,
				Code:    "REQUIRED",
				Message: fmt.Sprintf("%s is required", field),
			})
			continue
		}

		if !exists {
			continue
		}

		// Type validation
		if err := s.validateType(field, value, rule.Type); err != nil {
			errs = append(errs, *err)
			continue
		}

		// Range validation
		if err := s.validateRange(field, value, rule.Min, rule.Max); err != nil {
			errs = append(errs, *err)
		}

		// Pattern validation
		if rule.Pattern != nil {
			if str, ok := value.(string); ok {
				if !rule.Pattern.MatchString(str) {
					errs = append(errs, ValidationError{
						Field:   field,
						Code:    "PATTERN",
						Message: fmt.Sprintf("%s does not match required pattern", field),
					})
				}
			}
		}

		// OneOf validation
		if len(rule.OneOf) > 0 {
			if !contains(rule.OneOf, value) {
				errs = append(errs, ValidationError{
					Field:   field,
					Code:    "ONE_OF",
					Message: fmt.Sprintf("%s must be one of: %v", field, rule.OneOf),
				})
			}
		}

		// Custom validation
		if rule.Custom != nil {
			if err := rule.Custom(value); err != nil {
				errs = append(errs, ValidationError{
					Field:   field,
					Code:    "CUSTOM",
					Message: err.Error(),
				})
			}
		}
	}

	return errs
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Code    string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func (s *Schema) validateType(field string, value interface{}, expectedType string) *ValidationError {
	if expectedType == "" {
		return nil
	}

	actualType := reflect.TypeOf(value).Kind().String()

	switch expectedType {
	case "string":
		if actualType != "string" {
			return &ValidationError{
				Field:   field,
				Code:    "TYPE_MISMATCH",
				Message: fmt.Sprintf("%s must be a string, got %s", field, actualType),
			}
		}
	case "int":
		if actualType != "int" && actualType != "float64" {
			return &ValidationError{
				Field:   field,
				Code:    "TYPE_MISMATCH",
				Message: fmt.Sprintf("%s must be an integer, got %s", field, actualType),
			}
		}
	case "bool":
		if actualType != "bool" {
			return &ValidationError{
				Field:   field,
				Code:    "TYPE_MISMATCH",
				Message: fmt.Sprintf("%s must be a boolean, got %s", field, actualType),
			}
		}
	}

	return nil
}

func (s *Schema) validateRange(field string, value interface{}, min, max interface{}) *ValidationError {
	if min == nil && max == nil {
		return nil
	}

	var num float64
	switch v := value.(type) {
	case int:
		num = float64(v)
	case float64:
		num = v
	case string:
		if min == nil && max == nil {
			return nil
		}
		// For strings, check length
		length := float64(len(v))
		if min != nil {
			if minVal, ok := min.(int); ok && length < float64(minVal) {
				return &ValidationError{
					Field:   field,
					Code:    "MIN_LENGTH",
					Message: fmt.Sprintf("%s must be at least %d characters", field, minVal),
				}
			}
		}
		if max != nil {
			if maxVal, ok := max.(int); ok && length > float64(maxVal) {
				return &ValidationError{
					Field:   field,
					Code:    "MAX_LENGTH",
					Message: fmt.Sprintf("%s must be at most %d characters", field, maxVal),
				}
			}
		}
		return nil
	default:
		return nil
	}

	if min != nil {
		if minVal, ok := min.(int); ok && num < float64(minVal) {
			return &ValidationError{
				Field:   field,
				Code:    "MIN_VALUE",
				Message: fmt.Sprintf("%s must be at least %d", field, minVal),
			}
		}
	}

	if max != nil {
		if maxVal, ok := max.(int); ok && num > float64(maxVal) {
			return &ValidationError{
				Field:   field,
				Code:    "MAX_VALUE",
				Message: fmt.Sprintf("%s must be at most %d", field, maxVal),
			}
		}
	}

	return nil
}

func contains(slice []interface{}, value interface{}) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
		// Also check string equality for flexibility
		if str1, ok1 := v.(string); ok1 {
			if str2, ok2 := value.(string); ok2 {
				if str1 == str2 {
					return true
				}
			}
		}
	}
	return false
}

// ConfigValidator validates GLive configuration
type ConfigValidator struct {
	schema *Schema
}

// NewConfigValidator creates a new config validator with predefined rules
func NewConfigValidator() *ConfigValidator {
	schema := NewSchema().
		AddRule(ValidationRule{
			Field:    "api-key",
			Required: false,
			Type:     "string",
			Min:      20,
			Max:      200,
			Pattern:  regexp.MustCompile(`^[a-zA-Z0-9_-]+$`),
		}).
		AddRule(ValidationRule{
			Field:    "api-provider",
			Required: false,
			Type:     "string",
			OneOf:    []interface{}{"deepseek", "openai", "claude"},
		}).
		AddRule(ValidationRule{
			Field:    "workspace-dir",
			Required: false,
			Type:     "string",
			Custom: func(v interface{}) error {
				path := v.(string)
				if strings.Contains(path, "..") {
					return fmt.Errorf("path traversal not allowed")
				}
				return nil
			},
		}).
		AddRule(ValidationRule{
			Field:    "agent-port",
			Required: false,
			Type:     "int",
			Min:      1024,
			Max:      65535,
		}).
		AddRule(ValidationRule{
			Field:    "max-concurrent",
			Required: false,
			Type:     "int",
			Min:      1,
			Max:      100,
		}).
		AddRule(ValidationRule{
			Field:   "default-mode",
			Required: false,
			Type:    "string",
			OneOf:   []interface{}{"auto", "assisted", "manual"},
		}).
		AddRule(ValidationRule{
			Field:   "enable-sandbox",
			Required: false,
			Type:    "bool",
		})

	return &ConfigValidator{schema: schema}
}

// Validate validates a configuration map
func (v *ConfigValidator) Validate(config map[string]interface{}) error {
	errs := v.schema.Validate(config)
	if len(errs) > 0 {
		messages := make([]string, len(errs))
		for i, e := range errs {
			messages[i] = e.Error()
		}
		return errors.NewUserError("CONFIG_INVALID",
			"Configuration validation failed:\n  - "+strings.Join(messages, "\n  - "))
	}
	return nil
}

