package prompts

import (
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

// TemplateValidator validates prompt templates
type TemplateValidator struct {
	manager *PromptManager
}

// NewTemplateValidator creates a new template validator
func NewTemplateValidator(manager *PromptManager) *TemplateValidator {
	return &TemplateValidator{
		manager: manager,
	}
}

// ValidateTemplate validates a template structure and content
func (tv *TemplateValidator) ValidateTemplate(tmpl *PromptTemplate) error {
	var errors []string

	// Check required fields
	if tmpl.Name == "" {
		errors = append(errors, "template name is required")
	}

	if tmpl.Template == "" {
		errors = append(errors, "template content is required")
	}

	// Validate template syntax
	if err := tv.validateTemplateSyntax(tmpl.Template); err != nil {
		errors = append(errors, fmt.Sprintf("template syntax error: %v", err))
	}

	// Validate variables match template
	if err := tv.validateVariables(tmpl); err != nil {
		errors = append(errors, fmt.Sprintf("variable validation error: %v", err))
	}

	// Validate constraints
	if err := tv.validateConstraints(&tmpl.Constraints); err != nil {
		errors = append(errors, fmt.Sprintf("constraints validation error: %v", err))
	}

	// Validate parent reference if present
	if tmpl.Parent != "" {
		if _, ok := tv.manager.GetTemplate(tmpl.Parent); !ok {
			errors = append(errors, fmt.Sprintf("parent template '%s' not found", tmpl.Parent))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// validateTemplateSyntax validates Go template syntax
func (tv *TemplateValidator) validateTemplateSyntax(templateStr string) error {
	_, err := template.New("validation").Parse(templateStr)
	return err
}

// validateVariables checks if declared variables match template usage
func (tv *TemplateValidator) validateVariables(tmpl *PromptTemplate) error {
	// Extract variables from template
	extracted := ExtractVariables(tmpl.Template)

	// Check if all extracted variables are declared
	declared := make(map[string]bool)
	for _, v := range tmpl.Variables {
		declared[v] = true
	}

	missing := []string{}
	for _, v := range extracted {
		if !declared[v] {
			missing = append(missing, v)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("undeclared variables found in template: %s", strings.Join(missing, ", "))
	}

	return nil
}

// validateConstraints validates prompt constraints
func (tv *TemplateValidator) validateConstraints(constraints *PromptConstraints) error {
	if constraints.MaxTokens < 0 {
		return fmt.Errorf("max_tokens must be non-negative")
	}

	if constraints.Temperature < 0 || constraints.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}

	if constraints.TopP < 0 || constraints.TopP > 1 {
		return fmt.Errorf("top_p must be between 0 and 1")
	}

	if constraints.FrequencyPen < -2 || constraints.FrequencyPen > 2 {
		return fmt.Errorf("frequency_penalty must be between -2 and 2")
	}

	if constraints.PresencePen < -2 || constraints.PresencePen > 2 {
		return fmt.Errorf("presence_penalty must be between -2 and 2")
	}

	if constraints.ResponseFormat != "" && constraints.ResponseFormat != "text" && constraints.ResponseFormat != "json" {
		return fmt.Errorf("response_format must be 'text' or 'json'")
	}

	return nil
}

// ValidateRenderInput validates variables before rendering
func (tv *TemplateValidator) ValidateRenderInput(tmpl *PromptTemplate, vars map[string]interface{}) error {
	// Check required variables
	missing := []string{}
	for _, varName := range tmpl.Variables {
		if _, ok := vars[varName]; !ok {
			missing = append(missing, varName)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required variables: %s", strings.Join(missing, ", "))
	}

	// Validate variable types (basic checks)
	for varName, value := range vars {
		if value == nil {
			continue
		}

		// Check for common issues
		switch v := value.(type) {
		case string:
			// Check for potential injection
			if tv.detectInjection(v) {
				return fmt.Errorf("potential injection detected in variable '%s'", varName)
			}
		}
	}

	return nil
}

// detectInjection detects potential prompt injection attempts
func (tv *TemplateValidator) detectInjection(value string) bool {
	valueLower := strings.ToLower(value)
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
		if strings.Contains(valueLower, pattern) {
			return true
		}
	}

	return false
}

// ValidateTemplateName validates template name format
func ValidateTemplateName(name string) error {
	// Template names should be alphanumeric with underscores and hyphens
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	if !matched {
		return fmt.Errorf("template name must contain only alphanumeric characters, underscores, and hyphens")
	}

	if len(name) == 0 {
		return fmt.Errorf("template name cannot be empty")
	}

	if len(name) > 100 {
		return fmt.Errorf("template name cannot exceed 100 characters")
	}

	return nil
}

// ValidateVersion validates version format
func ValidateVersion(version string) error {
	// Remove "v" prefix if present
	version = strings.TrimPrefix(version, "v")

	// Should be numeric or semantic version
	matched, _ := regexp.MatchString(`^(\d+|[0-9]+\.[0-9]+\.[0-9]+)$`, version)
	if !matched {
		return fmt.Errorf("version must be numeric (e.g., '1', '2') or semantic (e.g., '1.0.0')")
	}

	return nil
}


