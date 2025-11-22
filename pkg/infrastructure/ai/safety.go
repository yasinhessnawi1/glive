package ai

import (
	"regexp"
	"strings"
)

// SafetyFilter filters out potential prompt injection attempts
type SafetyFilter struct {
	patterns []*regexp.Regexp
}

// NewSafetyFilter creates a new safety filter with default patterns
func NewSafetyFilter() *SafetyFilter {
	return &SafetyFilter{
		patterns: []*regexp.Regexp{
			// Ignore previous instructions
			regexp.MustCompile(`(?i)ignore\s+(previous|above|all)\s+instructions`),
			regexp.MustCompile(`(?i)disregard\s+.*instructions`),
			regexp.MustCompile(`(?i)forget\s+(all|previous|above)`),
			
			// Role-playing attempts
			regexp.MustCompile(`(?i)you\s+are\s+now`),
			regexp.MustCompile(`(?i)pretend\s+to\s+be`),
			regexp.MustCompile(`(?i)act\s+as\s+if`),
			regexp.MustCompile(`(?i)roleplay\s+as`),
			
			// New instructions
			regexp.MustCompile(`(?i)new\s+instructions?:`),
			regexp.MustCompile(`(?i)override\s+instructions`),
			regexp.MustCompile(`(?i)replace\s+instructions`),
			
			// System prompt injection
			regexp.MustCompile(`(?i)system\s*:\s*`),
			regexp.MustCompile(`(?i)assistant\s*:\s*`),
			regexp.MustCompile(`(?i)user\s*:\s*`),
			
			// Bypass attempts
			regexp.MustCompile(`(?i)bypass\s+security`),
			regexp.MustCompile(`(?i)ignore\s+security`),
			
			// Command execution attempts
			regexp.MustCompile(`(?i)execute\s+this\s+command`),
			regexp.MustCompile(`(?i)run\s+this\s+code`),
			regexp.MustCompile(`(?i)do\s+not\s+follow`),
			
			// Hidden instructions
			regexp.MustCompile(`(?i)\[INST\]`),
			regexp.MustCompile(`(?i)\[/INST\]`),
			regexp.MustCompile(`(?i)<\|im_start\|>`),
			regexp.MustCompile(`(?i)<\|im_end\|>`),
		},
	}
}

// SanitizeInput removes potential prompt injection attempts from content
func (f *SafetyFilter) SanitizeInput(content string) string {
	sanitized := content
	
	for _, pattern := range f.patterns {
		sanitized = pattern.ReplaceAllString(sanitized, "[FILTERED]")
	}
	
	return sanitized
}

// DetectInjection detects if content contains prompt injection attempts
func (f *SafetyFilter) DetectInjection(content string) bool {
	contentLower := strings.ToLower(content)
	
	for _, pattern := range f.patterns {
		if pattern.MatchString(contentLower) {
			return true
		}
	}
	
	return false
}

// SanitizeReadme sanitizes README content before sending to AI
func (f *SafetyFilter) SanitizeReadme(readmeContent string) string {
	return f.SanitizeInput(readmeContent)
}

// SanitizeFileList sanitizes file list content
func (f *SafetyFilter) SanitizeFileList(fileList []string) []string {
	sanitized := make([]string, len(fileList))
	for i, file := range fileList {
		sanitized[i] = f.SanitizeInput(file)
	}
	return sanitized
}

// AddPattern adds a custom pattern to the filter
func (f *SafetyFilter) AddPattern(pattern string) error {
	compiled, err := regexp.Compile(`(?i)` + pattern)
	if err != nil {
		return err
	}
	f.patterns = append(f.patterns, compiled)
	return nil
}

// SanitizeAnalysisInput sanitizes an entire AnalysisInput
func (f *SafetyFilter) SanitizeAnalysisInput(input AnalysisInput) AnalysisInput {
	return AnalysisInput{
		ProjectPath:   input.ProjectPath, // Paths are usually safe
		ReadmeContent: f.SanitizeReadme(input.ReadmeContent),
		FileList:      f.SanitizeFileList(input.FileList),
	}
}

