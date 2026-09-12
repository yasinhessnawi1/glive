package prompts

import (
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"unicode"
)

// TemplateEngine provides advanced template rendering capabilities
type TemplateEngine struct {
	manager *PromptManager
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine(manager *PromptManager) *TemplateEngine {
	return &TemplateEngine{
		manager: manager,
	}
}

// RenderWithInheritance renders a template with inheritance support
func (te *TemplateEngine) RenderWithInheritance(templateID string, vars map[string]interface{}) (string, error) {
	tmpl, ok := te.manager.GetTemplate(templateID)
	if !ok {
		return "", fmt.Errorf("template %s not found", templateID)
	}

	// Resolve inheritance
	resolved, err := te.manager.resolveTemplate(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to resolve template inheritance: %w", err)
	}

	// Render template
	return te.renderTemplate(resolved, vars)
}

// RenderWithSystemPrompt renders a template with its system prompt
func (te *TemplateEngine) RenderWithSystemPrompt(templateID string, vars map[string]interface{}) (string, string, error) {
	tmpl, ok := te.manager.GetTemplate(templateID)
	if !ok {
		return "", "", fmt.Errorf("template %s not found", templateID)
	}

	// Resolve inheritance
	resolved, err := te.manager.resolveTemplate(tmpl)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve template inheritance: %w", err)
	}

	// Render system prompt if present
	systemPrompt := ""
	if resolved.SystemPrompt != "" {
		systemPrompt, err = te.renderTemplate(&PromptTemplate{
			Template: resolved.SystemPrompt,
		}, vars)
		if err != nil {
			return "", "", fmt.Errorf("failed to render system prompt: %w", err)
		}
	}

	// Render user prompt
	userPrompt, err := te.renderTemplate(resolved, vars)
	if err != nil {
		return "", "", err
	}

	return systemPrompt, userPrompt, nil
}

// renderTemplate renders a template with variables
func (te *TemplateEngine) renderTemplate(tmpl *PromptTemplate, vars map[string]interface{}) (string, error) {
	// Create template with helper functions
	t, err := template.New(tmpl.ID).Funcs(te.getTemplateFuncs()).Parse(tmpl.Template)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder
	if err := t.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return buf.String(), nil
}

// titleCase upper-cases the first letter of each whitespace-separated word.
//
// It replaces strings.Title, which is deprecated because its word-boundary rule
// mishandles Unicode punctuation. The documented alternative is
// golang.org/x/text/cases, but this is a cosmetic helper exposed to prompt
// templates - pulling in a new module for it would not meet the "can this be
// written in a few lines?" bar in the dependency standards.
func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		r := []rune(w)
		r[0] = unicode.ToUpper(r[0])
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}

// getTemplateFuncs returns helper functions for templates
func (te *TemplateEngine) getTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"upper":      strings.ToUpper,
		"lower":      strings.ToLower,
		"title":      titleCase,
		"trim":       strings.TrimSpace,
		"join":       strings.Join,
		"contains":   strings.Contains,
		"hasPrefix":  strings.HasPrefix,
		"hasSuffix":  strings.HasSuffix,
		"replace":    strings.Replace,
		"replaceAll": strings.ReplaceAll,
		"default": func(def, val interface{}) interface{} {
			if val == nil || val == "" {
				return def
			}
			return val
		},
		"json": func(v interface{}) string {
			// Simple JSON encoding (for basic types)
			return fmt.Sprintf("%v", v)
		},
	}
}

// ValidateVariables checks if all required variables are provided
func (te *TemplateEngine) ValidateVariables(tmpl *PromptTemplate, vars map[string]interface{}) error {
	missing := []string{}
	for _, varName := range tmpl.Variables {
		if _, ok := vars[varName]; !ok {
			missing = append(missing, varName)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

// ExtractVariables extracts variable names from a template string
func ExtractVariables(templateStr string) []string {
	// Simple extraction using regex
	// Matches {{.variable_name}} patterns
	re := regexp.MustCompile(`\{\{\.(\w+)\}\}`)
	matches := re.FindAllStringSubmatch(templateStr, -1)

	variables := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			variables[match[1]] = true
		}
	}

	result := make([]string, 0, len(variables))
	for v := range variables {
		result = append(result, v)
	}

	return result
}
