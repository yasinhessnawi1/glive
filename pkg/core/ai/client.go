package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Client handles AI API interactions
type Client struct {
	apiKey           string
	provider         string
	endpoint         string
	httpClient       *http.Client
	classifier       *ErrorClassifier
	triedStrategies  map[string][]string // Track strategies tried per command
}

// NewClient creates a new AI client
func NewClient(apiKey, provider, endpoint string) *Client {
	if endpoint == "" {
		endpoint = getDefaultEndpoint(provider)
	}

	return &Client{
		apiKey:   apiKey,
		provider: provider,
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		classifier:      NewErrorClassifier(),
		triedStrategies: make(map[string][]string),
	}
}

// getDefaultEndpoint returns the default API endpoint for a provider
func getDefaultEndpoint(provider string) string {
	switch provider {
	case "deepseek":
		return "https://api.deepseek.com/v1/chat/completions"
	case "openai":
		return "https://api.openai.com/v1/chat/completions"
	case "claude":
		return "https://api.anthropic.com/v1/messages"
	default:
		return "https://api.deepseek.com/v1/chat/completions"
	}
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents an API request
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// ChatResponse represents an API response
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Chat sends a chat request to the AI API
func (c *Client) Chat(messages []Message) (string, error) {
	return c.ChatWithContext(context.Background(), messages)
}

// ChatWithContext sends a chat request to the AI API with context support
func (c *Client) ChatWithContext(ctx context.Context, messages []Message) (string, error) {
	reqBody := ChatRequest{
		Model:       c.getModel(),
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   4000,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Check if it's a context timeout
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("AI analysis timed out - the project may be too large or complex")
		}
		return "", fmt.Errorf("network error - please check your internet connection: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for better error messages
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", c.handleAPIError(resp.StatusCode, body)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// handleAPIError provides user-friendly error messages based on status code
func (c *Client) handleAPIError(statusCode int, body []byte) error {
	bodyStr := string(body)

	switch statusCode {
	case 400:
		return fmt.Errorf("invalid request to AI API - please check your configuration")
	case 401:
		return fmt.Errorf("invalid API key - please run: glive config set api-key YOUR_KEY")
	case 402:
		return fmt.Errorf("insufficient credits - please add funds to your %s account", c.provider)
	case 403:
		return fmt.Errorf("access forbidden - your API key may not have permission")
	case 429:
		return fmt.Errorf("rate limit exceeded - please wait a moment and try again")
	case 500, 502, 503:
		return fmt.Errorf("%s service temporarily unavailable - please try again later", c.provider)
	case 504:
		return fmt.Errorf("request timeout - the AI service took too long to respond")
	default:
		// Try to extract error message from response
		if len(bodyStr) > 0 && len(bodyStr) < 500 {
			return fmt.Errorf("AI API error (%d): %s", statusCode, bodyStr)
		}
		return fmt.Errorf("AI API error (status %d)", statusCode)
	}
}

// getModel returns the model name for the provider
func (c *Client) getModel() string {
	switch c.provider {
	case "deepseek":
		return "deepseek-chat"
	case "openai":
		return "gpt-4"
	case "claude":
		return "claude-3-sonnet-20240229"
	default:
		return "deepseek-chat"
	}
}

// AnalyzeProject uses AI to analyze a project
// Returns the raw JSON string which can be unmarshaled by the caller
func (c *Client) AnalyzeProject(ctx context.Context, projectPath string, readmeContent string, fileList []string) (string, error) {
	prompt := c.buildAnalysisPrompt(projectPath, readmeContent, fileList)

	messages := []Message{
		{
			Role:    "system",
			Content: "You are an expert software engineer analyzing GitHub projects. Provide detailed, accurate analysis in JSON format.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	response, err := c.ChatWithContext(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("AI analysis failed: %w", err)
	}

	return response, nil
}

// buildAnalysisPrompt creates the analysis prompt
func (c *Client) buildAnalysisPrompt(projectPath string, readmeContent string, fileList []string) string {
	return fmt.Sprintf(`Analyze this GitHub project and provide setup commands that will work on Windows.

Project Path: %s

README Content:
%s

File Structure:
%v

Respond with JSON:
{
  "project_type": "nodejs|python|go|rust|java|docker|polyglot|unknown",
  "detected_languages": ["language1", "language2"],
  "package_managers": ["npm", "pip", etc],
  "entry_points": ["main file paths"],
  "dependencies": [{"name": "dep", "version": "1.0", "type": "package", "installed": false}],
  "system_requirements": ["node>=16", "python>=3.8"],
  "commands": [
    {
      "id": "cmd-1",
      "description": "What this command does",
      "command": "the actual command",
      "working_dir": "%s",
      "stage": "setup|build|run",
      "required": true,
      "status": "pending"
    }
  ],
  "is_suspicious": false,
  "suspicious_reasons": [],
  "estimated_size": "50MB",
  "description": "Brief project description"
}

RULES:
1. Set "working_dir" to "%s" for ALL commands (no cd commands)
2. Each command must be directly executable (executable + args only)
3. Use the correct interpreter for each file type based on its extension
4. Generate commands appropriate for Windows
5. Think step-by-step: what would a developer need to do to run this project?

Provide ONLY valid JSON.`, projectPath, readmeContent, fileList, projectPath, projectPath)
}

// DebugError uses AI to debug an error
func (c *Client) DebugError(command string, output string, errorMsg string) (string, error) {
	prompt := fmt.Sprintf(`A command failed during project setup. Help debug and suggest a fix.

Command: %s
Output: %s
Error: %s

Provide a clear explanation of the error and specific steps to fix it.`, command, output, errorMsg)

	messages := []Message{
		{
			Role:    "system",
			Content: "You are an expert debugger. Provide clear, actionable solutions to errors.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	return c.Chat(messages)
}

// AutoFixError attempts to automatically fix a command error
// Returns: (fixed_command, explanation, can_auto_fix, error)
func (c *Client) AutoFixError(command string, output string, errorMsg string, workingDir string) (string, string, bool, error) {
	// Get file structure to provide context
	fileList := getProjectFiles(workingDir)
	fileListStr := strings.Join(fileList, "\n")

	// Track what has been tried before for this command
	cmdKey := command + ":" + errorMsg
	tried := c.triedStrategies[cmdKey]
	triedStr := ""
	if len(tried) > 0 {
		triedStr = fmt.Sprintf("\nPreviously tried strategies (DON'T repeat these):\n%s\n", strings.Join(tried, "\n- "))
	}

	prompt := fmt.Sprintf(`A command failed. Suggest a DIFFERENT recovery strategy than what was tried before.

Command: %s
Working Directory: %s
Output: %s
Error: %s
%s

Project Files (for context - DO NOT assume files are missing if they appear here):
%s

Respond with JSON:
{
  "can_auto_fix": true/false,
  "fixed_command": "corrected command (only if can_auto_fix is true)",
  "explanation": "What was wrong and how you fixed it",
  "confidence": "high|medium|low",
  "strategy_type": "one of: fix_syntax, change_approach, skip_optional, use_alternative, modify_env, install_dependency, retry, other"
}

Recovery Strategy Guidelines (try different approaches each time):
1. Fix syntax/typos in the command
2. Use alternative commands (e.g., 'cat' instead of 'copy', PowerShell instead of cmd)
3. Skip optional/non-critical commands (env file setup, linting, etc.)
4. Use development mode instead of production builds
5. Change environment variables or configuration
6. Install missing dependencies
7. Retry with delays for network issues
8. Use different package managers or tools

Rules:
- IMPORTANT: Provide a DIFFERENT strategy each time - don't repeat what was tried before
- If command is optional (env setup, config, linting), suggest skipping it entirely
- For file operations, try OS-specific alternatives (PowerShell on Windows, sh on Unix)
- Always check the "Project Files" list before assuming files are missing
- Be creative - try workarounds, alternatives, and fallbacks
- If all reasonable strategies exhausted, set can_auto_fix to false

Provide ONLY valid JSON.`, command, workingDir, output, errorMsg, triedStr, fileListStr)

	messages := []Message{
		{
			Role:    "system",
			Content: "You are an expert at diagnosing and fixing command-line errors. You are creative and persistent - always try to find an alternative solution. Suggest skipping non-critical commands if needed. Provide different strategies each time.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	response, err := c.Chat(messages)
	if err != nil {
		return "", "", false, fmt.Errorf("AI auto-fix analysis failed: %w", err)
	}

	// Parse the response
	var result struct {
		CanAutoFix   bool   `json:"can_auto_fix"`
		FixedCommand string `json:"fixed_command"`
		Explanation  string `json:"explanation"`
		Confidence   string `json:"confidence"`
		StrategyType string `json:"strategy_type"`
	}

	// Try to unmarshal
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Try to extract JSON
		start := strings.Index(response, "{")
		end := strings.LastIndex(response, "}")
		if start >= 0 && end > start {
			jsonStr := response[start : end+1]
			if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
				return "", "", false, fmt.Errorf("failed to parse AI auto-fix response: %w", err)
			}
		} else {
			return "", "", false, fmt.Errorf("no valid JSON in AI response")
		}
	}

	// Track this strategy attempt
	strategyDesc := fmt.Sprintf("%s: %s", result.StrategyType, result.Explanation)
	c.triedStrategies[cmdKey] = append(c.triedStrategies[cmdKey], strategyDesc)

	// Allow high/medium confidence fixes, or low confidence skip_optional strategies
	if result.CanAutoFix {
		if result.Confidence == "high" || result.Confidence == "medium" {
			return result.FixedCommand, result.Explanation, true, nil
		}
		// Allow low confidence if it's a skip strategy
		if result.StrategyType == "skip_optional" {
			return result.FixedCommand, result.Explanation, true, nil
		}
	}

	return "", result.Explanation, false, nil
}

// ClassifyError uses pattern matching and AI to classify an error
func (c *Client) ClassifyError(command, output, errorMsg string) (*ErrorContext, error) {
	// First try pattern matching (fast)
	ctx := c.classifier.Classify(command, output, errorMsg)

	// If confidence is high enough, return pattern-based classification
	if ctx.Confidence >= 0.7 {
		return ctx, nil
	}

	// For unknown or low-confidence errors, use AI classification
	prompt := fmt.Sprintf(`Classify this error into one of these categories:
- linting: ESLint, TypeScript, or other linting errors
- missing_dep: Missing packages or modules
- port_conflict: Port already in use
- permission: Access denied or permission errors
- network: Network timeouts or connection issues
- syntax: Code syntax errors
- build_failure: Build or compilation failures
- runtime: Runtime exceptions
- unknown: Cannot classify

Command: %s
Output: %s
Error: %s

Respond with JSON:
{
  "category": "one of the categories above",
  "severity": "critical|warning|info",
  "is_auto_fixable": true/false,
  "confidence": 0.0-1.0,
  "explanation": "brief explanation"
}

Provide ONLY valid JSON.`, command, output, errorMsg)

	messages := []Message{
		{
			Role:    "system",
			Content: "You are an expert at classifying software errors. Be precise and confident.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	response, err := c.Chat(messages)
	if err != nil {
		// Fallback to pattern-based classification
		return ctx, nil
	}

	// Parse AI response
	var aiResult struct {
		Category      string  `json:"category"`
		Severity      string  `json:"severity"`
		IsAutoFixable bool    `json:"is_auto_fixable"`
		Confidence    float64 `json:"confidence"`
		Explanation   string  `json:"explanation"`
	}

	// Try to unmarshal
	if err := json.Unmarshal([]byte(response), &aiResult); err != nil {
		// Try to extract JSON
		start := strings.Index(response, "{")
		end := strings.LastIndex(response, "}")
		if start >= 0 && end > start {
			jsonStr := response[start : end+1]
			if err := json.Unmarshal([]byte(jsonStr), &aiResult); err != nil {
				// Fallback to pattern-based
				return ctx, nil
			}
		} else {
			return ctx, nil
		}
	}

	// Merge AI result with pattern-based result
	if aiResult.Confidence > ctx.Confidence {
		ctx.Category = ErrorCategory(aiResult.Category)
		ctx.Severity = aiResult.Severity
		ctx.IsAutoFixable = aiResult.IsAutoFixable
		ctx.Confidence = aiResult.Confidence
	}

	return ctx, nil
}

// AutoFixWithStrategies attempts multiple recovery strategies
func (c *Client) AutoFixWithStrategies(command, output, errorMsg, workingDir string) ([]RecoveryStrategy, *ErrorContext, error) {
	// Classify the error
	ctx, err := c.ClassifyError(command, output, errorMsg)
	if err != nil {
		return nil, nil, err
	}

	// Get applicable strategies
	strategies := SelectStrategies(ctx)

	return strategies, ctx, nil
}

// getProjectFiles returns a list of files in the project directory
// This provides context to the AI so it knows what files exist
func getProjectFiles(workingDir string) []string {
	var files []string
	maxFiles := 200 // Limit to avoid overwhelming the AI

	// Walk the directory tree
	err := filepath.Walk(workingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip hidden directories and common ignore patterns
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") ||
				name == "node_modules" ||
				name == "__pycache__" ||
				name == "venv" ||
				name == "dist" ||
				name == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(workingDir, path)
		if err != nil {
			relPath = path
		}

		// Add to list if not too many files yet
		if len(files) < maxFiles {
			files = append(files, relPath)
		}

		return nil
	})

	if err != nil {
		return []string{"Error listing files: " + err.Error()}
	}

	if len(files) == 0 {
		return []string{"No files found"}
	}

	return files
}
