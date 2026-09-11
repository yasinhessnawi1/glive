package prompts

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

// PromptConstraints defines constraints for prompt execution
type PromptConstraints struct {
	MaxTokens      int      `yaml:"max_tokens"`
	Temperature    float64  `yaml:"temperature"`
	TopP           float64  `yaml:"top_p"`
	FrequencyPen   float64  `yaml:"frequency_penalty"`
	PresencePen    float64  `yaml:"presence_penalty"`
	ResponseFormat string   `yaml:"response_format"` // "text" or "json"
	StopSequences  []string `yaml:"stop_sequences"`
}

// PromptTemplate represents a prompt template loaded from YAML or text file
type PromptTemplate struct {
	ID           string            `yaml:"id"`
	Name         string            `yaml:"name"`
	Version      string            `yaml:"version"`
	Category     string            `yaml:"category"` // "analysis", "recovery", "security", "system"
	Template     string            `yaml:"template"`
	Variables    []string          `yaml:"variables"`
	Parent       string            `yaml:"parent"`        // Inherit from base template
	SystemPrompt string            `yaml:"system_prompt"` // Optional system prompt
	Constraints  PromptConstraints `yaml:"constraints"`
	FilePath     string            // Path to source file
	LastModified time.Time         // Last modification time
}

// PromptManager manages prompt templates with versioning and hot reload
type PromptManager struct {
	templates map[string]*PromptTemplate // key: "name:version" or "name" (latest)
	versions  map[string][]string        // key: name, value: list of versions
	basePath  string
	mu        sync.RWMutex
	hotReload bool
	watchers  map[string]time.Time // file path -> last check time
}

// NewPromptManager creates a new prompt manager
func NewPromptManager(basePath string) (*PromptManager, error) {
	pm := &PromptManager{
		templates: make(map[string]*PromptTemplate),
		versions:  make(map[string][]string),
		basePath:  basePath,
		watchers:  make(map[string]time.Time),
		hotReload: false, // Can be enabled via config
	}

	// Load all prompt templates
	if err := pm.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load prompt templates: %w", err)
	}

	return pm, nil
}

// EnableHotReload enables hot reload for prompt templates
func (pm *PromptManager) EnableHotReload() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.hotReload = true
}

// loadTemplates loads all prompt files from the base path (YAML and text files)
func (pm *PromptManager) loadTemplates() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Default prompts directory
	promptsDir := pm.basePath
	if promptsDir == "" {
		// Try to find prompts directory relative to current file
		promptsDir = filepath.Join("pkg", "infrastructure", "ai", "prompts")
	}

	// Load YAML templates from root directory (backward compatibility)
	promptFiles := []string{
		"analyze_project.yaml",
		"auto_fix.yaml",
		"debug_error.yaml",
	}

	for _, filename := range promptFiles {
		filePath := filepath.Join(promptsDir, filename)
		tmpl, err := pm.loadTemplate(filePath)
		if err != nil {
			// If file doesn't exist, that's okay - we'll use defaults
			continue
		}
		pm.registerTemplate(tmpl)
	}

	// Load templates from subdirectories
	templatesDir := filepath.Join(promptsDir, "templates")
	if err := pm.loadTemplatesFromDir(templatesDir); err != nil {
		// If templates directory doesn't exist, that's okay
		_ = err
	}

	return nil
}

// loadTemplatesFromDir recursively loads templates from a directory
func (pm *PromptManager) loadTemplatesFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			// Recursively load from subdirectories
			if err := pm.loadTemplatesFromDir(path); err != nil {
				continue
			}
		} else if strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml") || strings.HasSuffix(entry.Name(), ".txt") {
			tmpl, err := pm.loadTemplate(path)
			if err != nil {
				continue
			}
			pm.registerTemplate(tmpl)
		}
	}

	return nil
}

// registerTemplate registers a template with versioning support
func (pm *PromptManager) registerTemplate(tmpl *PromptTemplate) {
	// Extract version from filename if not set (e.g., "template_v2.txt" -> "2")
	if tmpl.Version == "" {
		tmpl.Version = pm.extractVersionFromFilename(tmpl.FilePath)
		if tmpl.Version == "" {
			tmpl.Version = "1" // Default to v1
		}
	}

	// Normalize version (remove "v" prefix if present)
	tmpl.Version = strings.TrimPrefix(tmpl.Version, "v")

	// Set ID if not set
	if tmpl.ID == "" {
		tmpl.ID = fmt.Sprintf("%s:%s", tmpl.Name, tmpl.Version)
	}

	// Register by ID
	pm.templates[tmpl.ID] = tmpl

	// Register as latest version if this is newer
	if versions, exists := pm.versions[tmpl.Name]; exists {
		pm.versions[tmpl.Name] = append(versions, tmpl.Version)
	} else {
		pm.versions[tmpl.Name] = []string{tmpl.Version}
		// Also register as latest
		pm.templates[tmpl.Name] = tmpl
	}

	// Track file for hot reload
	if tmpl.FilePath != "" {
		if info, err := os.Stat(tmpl.FilePath); err == nil {
			pm.watchers[tmpl.FilePath] = info.ModTime()
		}
	}
}

// extractVersionFromFilename extracts version from filename like "template_v2.txt"
func (pm *PromptManager) extractVersionFromFilename(filepath string) string {
	re := regexp.MustCompile(`_v(\d+)\.`)
	matches := re.FindStringSubmatch(filepath)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// loadTemplate loads a single prompt template from a YAML or text file
func (pm *PromptManager) loadTemplate(filePath string) (*PromptTemplate, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Get file info for modification time
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	var tmpl PromptTemplate

	// Try to parse as YAML first
	if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
		if err := yaml.Unmarshal(data, &tmpl); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	} else {
		// Text file - extract name and version from filename
		basename := filepath.Base(filePath)
		name := strings.TrimSuffix(basename, filepath.Ext(basename))
		// Remove version suffix if present
		name = regexp.MustCompile(`_v\d+$`).ReplaceAllString(name, "")

		tmpl.Name = name
		tmpl.Template = string(data)
		tmpl.Constraints.MaxTokens = 4000 // Default
		tmpl.Constraints.Temperature = 0.7
		tmpl.Constraints.ResponseFormat = "json"
	}

	// Set file path and modification time
	tmpl.FilePath = filePath
	tmpl.LastModified = info.ModTime()

	// Extract category from directory structure
	dir := filepath.Dir(filePath)
	if strings.Contains(dir, "analysis") {
		tmpl.Category = "analysis"
	} else if strings.Contains(dir, "recovery") {
		tmpl.Category = "recovery"
	} else if strings.Contains(dir, "security") {
		tmpl.Category = "security"
	} else if strings.Contains(dir, "system") {
		tmpl.Category = "system"
	}

	return &tmpl, nil
}

// Render renders a prompt template with the given variables
// Supports both "name" (latest version) and "name:version" formats
func (pm *PromptManager) Render(name string, vars map[string]interface{}) (string, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Check for hot reload
	if pm.hotReload {
		pm.checkForUpdates()
	}

	// Try to find template
	tmpl, ok := pm.templates[name]
	if !ok {
		return "", fmt.Errorf("prompt template %s not found", name)
	}

	// Handle template inheritance
	resolvedTemplate, err := pm.resolveTemplate(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to resolve template: %w", err)
	}

	// Parse the template
	t, err := template.New(tmpl.ID).Parse(resolvedTemplate.Template)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Render the template
	var buf strings.Builder
	if err := t.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return buf.String(), nil
}

// resolveTemplate resolves template inheritance
func (pm *PromptManager) resolveTemplate(tmpl *PromptTemplate) (*PromptTemplate, error) {
	if tmpl.Parent == "" {
		return tmpl, nil
	}

	// Find parent template
	parent, ok := pm.templates[tmpl.Parent]
	if !ok {
		return tmpl, nil // If parent not found, use template as-is
	}

	// Resolve parent first (recursive)
	resolvedParent, err := pm.resolveTemplate(parent)
	if err != nil {
		return tmpl, nil
	}

	// Merge: parent template + child template
	merged := *tmpl
	merged.Template = resolvedParent.Template + "\n\n" + tmpl.Template

	// Merge variables
	merged.Variables = append(resolvedParent.Variables, tmpl.Variables...)

	// Merge constraints (child overrides parent)
	if merged.Constraints.MaxTokens == 0 {
		merged.Constraints.MaxTokens = resolvedParent.Constraints.MaxTokens
	}
	if merged.Constraints.Temperature == 0 {
		merged.Constraints.Temperature = resolvedParent.Constraints.Temperature
	}

	return &merged, nil
}

// checkForUpdates checks for file updates and reloads if needed
func (pm *PromptManager) checkForUpdates() {
	pm.mu.RUnlock()
	defer pm.mu.RLock()

	for filePath, lastCheck := range pm.watchers {
		if info, err := os.Stat(filePath); err == nil {
			if info.ModTime().After(lastCheck) {
				// File was modified, reload it
				if tmpl, err := pm.loadTemplate(filePath); err == nil {
					pm.mu.Lock()
					pm.registerTemplate(tmpl)
					pm.mu.Unlock()
				}
			}
		}
	}
}

// GetTemplate returns a template by name (latest version) or "name:version"
func (pm *PromptManager) GetTemplate(name string) (*PromptTemplate, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	tmpl, ok := pm.templates[name]
	return tmpl, ok
}

// GetTemplateVersion returns a specific version of a template
func (pm *PromptManager) GetTemplateVersion(name, version string) (*PromptTemplate, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	id := fmt.Sprintf("%s:%s", name, version)
	tmpl, ok := pm.templates[id]
	return tmpl, ok
}

// ListVersions returns all versions of a template
func (pm *PromptManager) ListVersions(name string) []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.versions[name]
}

// RegisterTemplate registers a template programmatically
func (pm *PromptManager) RegisterTemplate(tmpl *PromptTemplate) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.registerTemplate(tmpl)
}

// Reload reloads all templates from disk
func (pm *PromptManager) Reload() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Clear existing templates
	pm.templates = make(map[string]*PromptTemplate)
	pm.versions = make(map[string][]string)
	pm.watchers = make(map[string]time.Time)

	// Reload
	return pm.loadTemplates()
}
