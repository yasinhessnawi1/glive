package ai

import (
	"strings"
	"unicode/utf8"
)

const (
	// MaxPromptTokens is the maximum number of tokens for prompts
	MaxPromptTokens = 8000
	// MaxResponseTokens is the maximum number of tokens for responses
	MaxResponseTokens = 4000
	// ApproximateCharsPerToken is an approximation: ~4 characters per token
	ApproximateCharsPerToken = 4
)

// TokenCounter counts and truncates tokens
type TokenCounter struct {
	charsPerToken int
}

// NewTokenCounter creates a new token counter
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{
		charsPerToken: ApproximateCharsPerToken,
	}
}

// Count estimates the number of tokens in a text string
func (t *TokenCounter) Count(text string) int {
	// Simple approximation: count characters and divide by chars per token
	// This is a rough estimate - for more accuracy, use tiktoken library
	charCount := utf8.RuneCountInString(text)
	return (charCount + t.charsPerToken - 1) / t.charsPerToken // Ceiling division
}

// Truncate truncates text to fit within maxTokens
func (t *TokenCounter) Truncate(text string, maxTokens int) string {
	if t.Count(text) <= maxTokens {
		return text
	}

	// Calculate max characters
	maxChars := maxTokens * t.charsPerToken

	// Truncate by runes to avoid breaking UTF-8 sequences
	runes := []rune(text)
	if len(runes) <= maxChars {
		return text
	}

	// Truncate and add ellipsis
	truncated := string(runes[:maxChars-3]) + "..."
	return truncated
}

// TruncateToPromptLimit truncates text to fit within prompt token limit
func (t *TokenCounter) TruncateToPromptLimit(text string) string {
	return t.Truncate(text, MaxPromptTokens)
}

// TruncateToResponseLimit truncates text to fit within response token limit
func (t *TokenCounter) TruncateToResponseLimit(text string) string {
	return t.Truncate(text, MaxResponseTokens)
}

// EstimateTokenCount estimates tokens for multiple strings
func (t *TokenCounter) EstimateTokenCount(texts ...string) int {
	total := 0
	for _, text := range texts {
		total += t.Count(text)
	}
	return total
}

// TruncateMultiple truncates multiple texts proportionally to fit within maxTokens
func (t *TokenCounter) TruncateMultiple(maxTokens int, texts ...string) []string {
	totalTokens := t.EstimateTokenCount(texts...)
	if totalTokens <= maxTokens {
		return texts
	}

	// Calculate truncation ratio
	ratio := float64(maxTokens) / float64(totalTokens)

	result := make([]string, len(texts))
	for i, text := range texts {
		textTokens := t.Count(text)
		targetTokens := int(float64(textTokens) * ratio)
		result[i] = t.Truncate(text, targetTokens)
	}

	return result
}

// TruncateReadme truncates README content if it's too long
func (t *TokenCounter) TruncateReadme(readmeContent string) string {
	// Reserve tokens for other parts of the prompt
	// Assume ~2000 tokens for prompt structure, file list, etc.
	readmeLimit := MaxPromptTokens - 2000
	return t.Truncate(readmeContent, readmeLimit)
}

// TruncateFileList truncates file list if it's too long
func (t *TokenCounter) TruncateFileList(fileList []string) []string {
	fileListStr := strings.Join(fileList, "\n")
	// Reserve tokens for other parts
	fileListLimit := MaxPromptTokens - 3000
	truncated := t.Truncate(fileListStr, fileListLimit)
	return strings.Split(truncated, "\n")
}

