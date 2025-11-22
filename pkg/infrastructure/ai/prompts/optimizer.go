package prompts

import (
	"fmt"
	"regexp"
	"strings"
)

// TokenOptimizer optimizes prompts to reduce token usage
type TokenOptimizer struct {
	tokenCounter func(string) int
}

// NewTokenOptimizer creates a new token optimizer
func NewTokenOptimizer(tokenCounter func(string) int) *TokenOptimizer {
	return &TokenOptimizer{
		tokenCounter: tokenCounter,
	}
}

// File represents a file with content
type File struct {
	Path    string
	Content string
}

// PruneFileContext prunes file context to only include relevant files
func (to *TokenOptimizer) PruneFileContext(files []File, relevantPaths []string, maxTokens int) string {
	var selected []File
	tokenCount := 0

	relevantSet := make(map[string]bool)
	for _, path := range relevantPaths {
		relevantSet[path] = true
	}

	for _, file := range files {
		if relevantSet[file.Path] && tokenCount < maxTokens {
			fileTokens := to.tokenCounter(file.Content)
			if tokenCount+fileTokens <= maxTokens {
				selected = append(selected, file)
				tokenCount += fileTokens
			}
		}
	}

	return to.formatFiles(selected)
}

// formatFiles formats files for inclusion in prompts
func (to *TokenOptimizer) formatFiles(files []File) string {
	var builder strings.Builder
	for _, file := range files {
		builder.WriteString(fmt.Sprintf("File: %s\n", file.Path))
		builder.WriteString(file.Content)
		builder.WriteString("\n\n")
	}
	return builder.String()
}

// TruncateOutput truncates command output to last N lines
func (to *TokenOptimizer) TruncateOutput(output string, maxLines int) string {
	lines := strings.Split(output, "\n")
	if len(lines) <= maxLines {
		return output
	}

	// Take first 20 and last N-20 lines
	firstLines := 20
	if maxLines < 40 {
		firstLines = maxLines / 2
	}

	first := lines[:firstLines]
	last := lines[len(lines)-(maxLines-firstLines):]

	var builder strings.Builder
	builder.WriteString(strings.Join(first, "\n"))
	builder.WriteString("\n... [truncated] ...\n")
	builder.WriteString(strings.Join(last, "\n"))

	return builder.String()
}

// CompressFrameworkOutput compresses verbose framework output
func (to *TokenOptimizer) CompressFrameworkOutput(output string) string {
	// Remove verbose webpack/vite build logs (percentage lines)
	output = regexp.MustCompile(`(?m)^\s*\d+%.*\n`).ReplaceAllString(output, "")

	// Compress dependency trees (tree characters)
	output = regexp.MustCompile(`(?m)^[│├└─\s]+.*\n`).ReplaceAllString(output, "")

	// Keep only errors and warnings
	lines := strings.Split(output, "\n")
	filtered := make([]string, 0, len(lines))
	
	errorKeywords := []string{"error", "warn", "fail", "exception", "traceback", "fatal"}
	
	for _, line := range lines {
		lineLower := strings.ToLower(line)
		for _, keyword := range errorKeywords {
			if strings.Contains(lineLower, keyword) {
				filtered = append(filtered, line)
				break
			}
		}
	}

	if len(filtered) == 0 {
		// If no errors/warnings, return last 50 lines
		if len(lines) > 50 {
			return strings.Join(lines[len(lines)-50:], "\n")
		}
		return output
	}

	return strings.Join(filtered, "\n")
}

// OptimizePrompt optimizes a prompt by removing unnecessary content
func (to *TokenOptimizer) OptimizePrompt(prompt string, maxTokens int) string {
	currentTokens := to.tokenCounter(prompt)
	if currentTokens <= maxTokens {
		return prompt
	}

	// Try to compress common verbose sections
	lines := strings.Split(prompt, "\n")
	optimized := make([]string, 0, len(lines))

	for _, line := range lines {
		// Skip empty lines (but keep structure)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			optimized = append(optimized, "")
			continue
		}

		// Skip verbose comments
		if strings.HasPrefix(trimmed, "#") && len(trimmed) > 100 {
			continue
		}

		optimized = append(optimized, line)
	}

	result := strings.Join(optimized, "\n")
	
	// If still too long, truncate
	if to.tokenCounter(result) > maxTokens {
		ratio := float64(maxTokens) / float64(to.tokenCounter(result))
		targetChars := int(float64(len(result)) * ratio)
		if targetChars < len(result) {
			result = result[:targetChars] + "\n... [truncated]"
		}
	}

	return result
}

// RemoveRedundantWhitespace removes redundant whitespace from text
func (to *TokenOptimizer) RemoveRedundantWhitespace(text string) string {
	// Replace multiple spaces with single space
	text = regexp.MustCompile(` +`).ReplaceAllString(text, " ")
	
	// Replace multiple newlines with double newline max
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	
	return strings.TrimSpace(text)
}

// ExtractKeyInformation extracts only key information from verbose output
func (to *TokenOptimizer) ExtractKeyInformation(output string) string {
	lines := strings.Split(output, "\n")
	keyLines := make([]string, 0)

	for _, line := range lines {
		lineLower := strings.ToLower(line)
		
		// Keep lines with important keywords
		importantKeywords := []string{
			"error", "warning", "fail", "success", "complete",
			"installed", "missing", "not found", "exception",
			"traceback", "fatal", "critical",
		}

		for _, keyword := range importantKeywords {
			if strings.Contains(lineLower, keyword) {
				keyLines = append(keyLines, line)
				break
			}
		}
	}

	if len(keyLines) == 0 {
		// If no key lines, return last 20 lines
		if len(lines) > 20 {
			return strings.Join(lines[len(lines)-20:], "\n")
		}
		return output
	}

	return strings.Join(keyLines, "\n")
}

