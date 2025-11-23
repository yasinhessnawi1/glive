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
	"runtime"
	"strings"

	securehttp "github.com/glive/infrastructure/http"
)

// Client handles AI API interactions
type Client struct {
	apiKey     string
	provider   string
	endpoint   string
	httpClient *http.Client
	classifier *ErrorClassifier
}

// NewClient creates a new AI client
func NewClient(apiKey, provider, endpoint string) *Client {
	if endpoint == "" {
		endpoint = getDefaultEndpoint(provider)
	}

	return &Client{
		apiKey:     apiKey,
		provider:   provider,
		endpoint:   endpoint,
		httpClient: securehttp.OptimizedClient(),
		classifier: NewErrorClassifier(),
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

// AnalyzeProject uses AI to analyze a project
// Returns the raw JSON string which can be unmarshaled by the caller
func (c *Client) AnalyzeProject(ctx context.Context, projectPath string, readmeContent string, fileList []string) (string, error) {
	prompt := c.buildAnalysisPrompt(projectPath, readmeContent, fileList)

	messages := []Message{
		{
			Role:    "system",
			Content: "You are an expert software architect and DevOps engineer. Your goal is to analyze projects and provide precise setup instructions.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	return c.ChatWithContext(ctx, messages)
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

// buildAnalysisPrompt creates the analysis prompt with full context
func (c *Client) buildAnalysisPrompt(projectPath string, readmeContent string, fileList []string) string {
	// Detect platform
	platform := c.detectPlatform()

	// Read configuration files for context
	configFiles := c.readConfigFiles(projectPath)

	// Build comprehensive prompt
	return fmt.Sprintf(`Analyze this GitHub project and provide setup commands that will work on %s.

## Project Path
%s

## README Content (IMPORTANT - Read this carefully!)
The README below contains the project's official setup instructions. Extract and follow these
instructions when generating commands. If the README specifies a particular setup process,
use that process instead of making assumptions.

%s

## File Structure
%v

## Configuration Files Detected
The following configuration files were found in the project. Use these to understand
the project's dependencies and build process:

%s

## Analysis Instructions

1. **First, read the README thoroughly** - Look for:
   - Installation/Setup sections
   - Prerequisites/Requirements
   - Getting Started guides
   - Build commands
   - Run commands

2. **Identify the project type** from file structure:
   - package.json → Node.js (npm/yarn/pnpm)
   - requirements.txt/Pipfile/pyproject.toml → Python
   - go.mod → Go
   - Cargo.toml → Rust
   - pom.xml/build.gradle → Java
   - Dockerfile/docker-compose.yml → Docker

3. **Extract commands from README** - If the README says "run npm install && npm start",
   generate those exact commands.

4. **Use the configuration files** - Parse package.json scripts, requirements.txt
   dependencies, etc. to understand what the project needs.

5. **Generate a step-by-step setup plan** that:
   - Follows the README's instructions when available
   - Installs dependencies first
   - Builds the project if needed
   - Runs the project

## Response Format

Respond with ONLY valid JSON (no markdown code blocks, no extra text):

{
  "project_type": "nodejs|python|go|rust|java|docker|polyglot|unknown",
  "detected_languages": ["language1", "language2"],
  "package_managers": ["npm", "pip", "go", etc.],
  "entry_points": ["main file paths from config or README"],
  "readme_instructions": "Brief summary of what the README says about setup",
  "dependencies": [
    {
      "name": "dependency-name",
      "version": "version-if-known",
      "type": "runtime|dev|peer|optional",
      "installed": false
    }
  ],
  "system_requirements": ["node>=16", "python>=3.8"],
  "commands": [
    {
      "id": "cmd-1",
      "description": "What this command does (from README or inferred)",
      "command": "the actual command",
      "working_dir": "%s",
      "stage": "setup|build|run",
      "required": true,
      "status": "pending",
      "source": "readme|inferred|default"
    }
  ],
  "is_suspicious": false,
  "suspicious_reasons": [],
  "estimated_size": "50MB",
  "description": "Brief project description based on README",
  "notes": "Any important notes about the setup process",
  "usage_instructions": [
    "Step 1: Run npm start",
    "Step 2: Open http://localhost:3000"
  ],
  "key_milestones": [
    "Install dependencies",
    "Build project",
    "Start server"
  ]
}

## Rules

1. Set "working_dir" to "%s" for ALL commands (no cd commands)
2. Each command must be directly executable (executable + args only)
3. Use the correct interpreter for each file type based on its extension
4. Generate commands appropriate for %s
5. Mark "source" as "readme" if the command came from README, "inferred" if you derived it
6. If README specifies exact commands, use those commands verbatim
7. Think step-by-step: what would a developer need to do to run this project?

Provide ONLY valid JSON - no markdown, no code blocks, just raw JSON.`, platform, projectPath, readmeContent, fileList, configFiles, projectPath, projectPath, platform)
}

// detectPlatform returns the current operating system name
func (c *Client) detectPlatform() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows"
	case "darwin":
		return "macOS"
	case "linux":
		return "Linux"
	default:
		return runtime.GOOS
	}
}

// readConfigFiles reads common configuration files and returns their contents
func (c *Client) readConfigFiles(projectPath string) string {
	configFileNames := []string{
		"package.json",
		"requirements.txt",
		"pyproject.toml",
		"Pipfile",
		"go.mod",
		"Cargo.toml",
		"pom.xml",
		"build.gradle",
		"docker-compose.yml",
		"docker-compose.yaml",
		"Makefile",
		".env.example",
	}

	var configContents strings.Builder
	for _, fileName := range configFileNames {
		filePath := filepath.Join(projectPath, fileName)
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue // File doesn't exist, skip
		}

		// Truncate very large files to avoid token limits
		contentStr := string(content)
		if len(contentStr) > 3000 {
			contentStr = contentStr[:3000] + "\n... (truncated)"
		}

		configContents.WriteString(fmt.Sprintf("\n### %s\n```\n%s\n```\n", fileName, contentStr))
	}

	if configContents.Len() == 0 {
		return "No common configuration files found."
	}

	return configContents.String()
}

// DebugError uses AI to debug an error
func (c *Client) DebugError(command string, output string, errorMsg string) (string, error) {
	platform := c.detectPlatform()

	prompt := fmt.Sprintf(`A command failed during project setup. Help debug and suggest a fix.

## Failed Command
%s

## Command Output
%s

## Error Message
%s

## Platform
%s

## Your Analysis Should Include

1. **Error Explanation** - What does this error mean in simple terms?
2. **Root Cause** - Why did it happen?
3. **Solutions** - Step-by-step instructions to fix it
4. **Prevention** - How to avoid this in the future

Provide a clear, well-structured explanation.`, command, output, errorMsg, platform)

	messages := []Message{
		{
			Role: "system",
			Content: `You are an expert software engineer and debugger with 20+ years of experience.
Your explanations should be:
- Clear and concise
- Actionable with specific steps
- Educational (help the user learn)
- Platform-aware (consider Windows/Linux/macOS differences)`,
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	return c.Chat(messages)
}

// AutoFixResult contains the result of an auto-fix attempt
type AutoFixResult struct {
	CanFix               bool           `json:"can_fix"`
	FixType              string         `json:"fix_type"`
	FixedCommand         string         `json:"fixed_command"`
	Explanation          string         `json:"explanation"`
	Confidence           string         `json:"confidence"`
	Analysis             []string       `json:"step_by_step_analysis"`
	Alternatives         []string       `json:"if_this_fails_try"`
	FilesToCreate        []FileToCreate `json:"files_to_create"`
	ManualInstructions   []string       `json:"manual_instructions"`
	IsUserActionRequired bool           `json:"is_user_action_required"`
	SkipIfOptional       bool           `json:"skip_if_optional"`
	SkipReason           string         `json:"skip_reason"`
}

// FileToCreate represents a file that needs to be created as part of a fix
type FileToCreate struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// AutoFixError attempts to automatically fix a command error
// Returns: (fixed_command, explanation, can_auto_fix, error)
// AutoFixError attempts to automatically fix a command error using the Agent
// Returns: (fixed_command, explanation, can_auto_fix, error)
func (c *Client) AutoFixError(command string, output string, errorMsg string, workingDir string) (string, string, bool, error) {
	// Use the new Agent for auto-fixing
	agent := NewAgent(c, workingDir)

	goal := fmt.Sprintf(`Fix this command failure:
Command: %s
Error Output: %s
Error Message: %s

Investigate why it failed and fix it. If you cannot fix it automatically (e.g., missing API key), provide manual instructions.`, command, output, errorMsg)

	result, err := agent.Run(context.Background(), goal)
	if err != nil {
		return "", "", false, err
	}

	if !result.Success {
		// If agent failed but provided manual instructions, return them in the explanation
		if len(result.ManualInstructions) > 0 {
			explanation := "I cannot fix this automatically. Please follow these steps:\n"
			for i, step := range result.ManualInstructions {
				explanation += fmt.Sprintf("%d. %s\n", i+1, step)
			}
			return "", explanation, false, nil
		}
		return "", result.Message, false, nil
	}

	// If success, the message should contain the fix details
	// Note: The agent might have already applied the fix via tools (e.g. edit_file)
	// If the fix requires re-running the command, we return the command.
	// For now, we'll assume if success is true, the user should retry the original command or the agent provided a new one.
	// To keep compatibility with the existing interface, we might need to parse the agent's message for a command.
	// But since the agent can run commands itself, maybe we just return the original command to retry?
	// Let's assume the agent fixed the environment/files, so retrying the original command is the way to go.

	return command, result.Message, true, nil
}

// AutoFixWithStrategies attempts to fix an error using recovery strategies
// This method exists for compatibility with the executor.AIClient interface
func (c *Client) AutoFixWithStrategies(command, output, errorMsg, workingDir string) ([]RecoveryStrategy, *ErrorContext, error) {
	// Classify the error
	errorCtx := c.classifier.Classify(command, output, errorMsg)

	// Get applicable strategies
	strategies := SelectStrategies(errorCtx)

	return strategies, errorCtx, nil
}

// AutoFixErrorWithAttempts attempts to fix with tracking of previous attempts
// This allows for more creative solutions on subsequent attempts
func (c *Client) AutoFixErrorWithAttempts(command string, output string, errorMsg string, workingDir string, attemptNumber int, previousAttempts []string) (*AutoFixResult, error) {
	platform := c.detectPlatform()

	// Format previous attempts for the prompt
	prevAttemptsStr := "None yet - this is the first attempt"
	if len(previousAttempts) > 0 {
		prevAttemptsStr = strings.Join(previousAttempts, "\n- ")
		prevAttemptsStr = "- " + prevAttemptsStr
	}

	// Get directory context to help the AI see what files actually exist
	dirContext := c.getDirectoryContext(workingDir)

	prompt := fmt.Sprintf(`A command failed. This is attempt #%d. You MUST find a way to make progress.

## Failed Command
%s

## Working Directory
%s

## Directory Contents (Files actually present)
%s

## Error Output
%s

## Error Message
%s

## Platform
%s

## Previous Attempts (DO NOT REPEAT THESE)
%s

## Your Task

DO NOT just identify the problem. You must SOLVE IT with a workaround.

Step 1: What EXACTLY is failing? (be specific, not "command failed")
Step 2: Check the "Directory Contents" list. Does the file exist with a different name? Is it in a subdirectory?
Step 3: What are 3 DIFFERENT ways to work around this?
Step 4: Pick the BEST workaround that will work RIGHT NOW
Step 5: Generate the EXACT command to try

## IMPORTANT: Creative Workarounds

If the error is about:
- **Missing file (.env, config)**: Check Directory Contents first! If it exists with a different name, use that. If not, CREATE it.
- **Command not found**: Find an alternative (npx, yarn, python3, py, etc.)
- **Permission denied**: Use a different path or approach
- **Module not found**: Install it inline or find alternative
- **Network error**: Retry with different mirror or skip if optional
- **Path not found**: Create the path or use absolute path

## Response Format (ONLY valid JSON, no markdown):

{
  "step_by_step_analysis": [
    "Step 1: The exact error is...",
    "Step 2: I checked the file list and found...",
    "Step 3: Three workarounds are: (a)... (b)... (c)..."
  ],
  "can_fix": true,
  "fix_type": "workaround|alternative_command|create_file|modify_command|skip",
  "fixed_command": "the exact command to run",
  "explanation": "What this fix does and why it should work",
  "confidence": "high|medium|low",
  "if_this_fails_try": [
    "Alternative approach 1 with exact command",
    "Alternative approach 2 with exact command",
    "Alternative approach 3 with exact command"
  ],
  "files_to_create": [
    {
      "path": "path/to/file",
      "content": "file content here"
    }
  ],
  "skip_if_optional": false,
  "skip_reason": "Only if this step is truly optional"
}

## Rules

1. ALWAYS provide a fixed_command - never give up
2. if_this_fails_try MUST have 3 different approaches
3. Each approach must be DIFFERENT from previous attempts
4. If you need to create a file, put it in files_to_create
5. Be creative - developers find workarounds, so should you
6. USE THE DIRECTORY LISTING! Do not assume files exist if they are not in the list.

Provide ONLY valid JSON - no markdown, no code blocks, just raw JSON.`, attemptNumber, command, workingDir, dirContext, output, errorMsg, platform, prevAttemptsStr)

	messages := []Message{
		{
			Role: "system",
			Content: `You are an expert debugger with 20+ years of experience. Your job is to fix command failures
by trying CREATIVE WORKAROUNDS, not just obvious fixes.

CRITICAL RULES:
1. NEVER assume the problem - analyze step by step
2. ALWAYS check the provided file list before assuming a file is missing or present
3. NEVER give up on first try - suggest alternative approaches
4. NEVER just say "install X" - provide a workaround that works NOW
5. If copying fails, CREATE the file with required content
6. If a command doesn't exist, find an ALTERNATIVE command
7. If permissions fail, try a different approach entirely
8. Think like a developer who MUST make this work

WORKAROUND STRATEGIES (use these!):
- File not found? → CHECK THE FILE LIST! Is it a typo? Is it in a subdir?
- Can't copy .env? → Create .env with echo/printf or write the vars directly
- npm not found? → Try npx, or check if yarn/pnpm exists
- Permission denied? → Try different directory, or use user-local paths
- Module not found? → Try installing it, or find alternative module
- Port in use? → Try different port
- Path issues? → Use absolute paths, or fix path separators`,
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	response, err := c.Chat(messages)
	if err != nil {
		return nil, fmt.Errorf("AI auto-fix analysis failed: %w", err)
	}

	// Parse the response
	var result AutoFixResult

	// Try to unmarshal directly first
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Try to extract JSON from response
		start := strings.Index(response, "{")
		end := strings.LastIndex(response, "}")
		if start >= 0 && end > start {
			jsonStr := response[start : end+1]
			if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
				// If we still can't parse, create a basic result
				return &AutoFixResult{
					CanFix:      false,
					Explanation: "Failed to parse AI response: " + err.Error(),
				}, nil
			}
		} else {
			return &AutoFixResult{
				CanFix:      false,
				Explanation: "No valid JSON in AI response",
			}, nil
		}
	}

	// Always try to fix unless explicitly marked as skip
	if result.FixedCommand == "" && len(result.Alternatives) > 0 {
		result.FixedCommand = result.Alternatives[0]
		result.CanFix = true
	}

	return &result, nil
}

// GetAlternativeFix returns an alternative fix from the result
func (r *AutoFixResult) GetAlternativeFix(index int) string {
	if index < len(r.Alternatives) {
		return r.Alternatives[index]
	}
	return ""
}

// PersistentFixConfig configures the persistent fix behavior
type PersistentFixConfig struct {
	MaxAttempts     int  // Maximum number of fix attempts (default: 10)
	TryAlternatives bool // Whether to try alternatives from AI suggestions
}

// DefaultPersistentFixConfig returns default configuration
func DefaultPersistentFixConfig() PersistentFixConfig {
	return PersistentFixConfig{
		MaxAttempts:     10,
		TryAlternatives: true,
	}
}

// PersistentFixSession tracks a multi-attempt fix session
type PersistentFixSession struct {
	client           *Client
	config           PersistentFixConfig
	command          string
	workingDir       string
	attemptNumber    int
	previousAttempts []string
	lastResult       *AutoFixResult
}

// NewPersistentFixSession creates a new persistent fix session
func (c *Client) NewPersistentFixSession(command, workingDir string, config PersistentFixConfig) *PersistentFixSession {
	return &PersistentFixSession{
		client:           c,
		config:           config,
		command:          command,
		workingDir:       workingDir,
		attemptNumber:    0,
		previousAttempts: []string{},
	}
}

// NextFix gets the next fix attempt
// Returns nil when max attempts reached or no more alternatives
func (s *PersistentFixSession) NextFix(errorOutput, errorMsg string) (*AutoFixResult, error) {
	s.attemptNumber++

	// Check if we've exceeded max attempts
	if s.attemptNumber > s.config.MaxAttempts {
		return nil, fmt.Errorf("max fix attempts (%d) exceeded", s.config.MaxAttempts)
	}

	// If we have a previous result with alternatives, try those first
	if s.lastResult != nil && s.config.TryAlternatives {
		altIndex := s.attemptNumber - 2 // -1 for first attempt, -1 for 0-index
		if altIndex >= 0 && altIndex < len(s.lastResult.Alternatives) {
			alt := s.lastResult.Alternatives[altIndex]
			s.previousAttempts = append(s.previousAttempts, fmt.Sprintf("Attempt %d: Tried alternative - %s", s.attemptNumber-1, alt))
			return &AutoFixResult{
				CanFix:       true,
				FixType:      "alternative_command",
				FixedCommand: alt,
				Explanation:  fmt.Sprintf("Trying alternative approach %d from previous suggestions", altIndex+1),
				Confidence:   "medium",
			}, nil
		}
	}

	// Record the previous attempt
	if s.attemptNumber > 1 && s.lastResult != nil {
		s.previousAttempts = append(s.previousAttempts, fmt.Sprintf("Attempt %d: %s - Result: Failed with error: %s",
			s.attemptNumber-1, s.lastResult.FixedCommand, truncateString(errorMsg, 100)))
	}

	// Get a new fix from AI
	result, err := s.client.AutoFixErrorWithAttempts(
		s.command,
		errorOutput,
		errorMsg,
		s.workingDir,
		s.attemptNumber,
		s.previousAttempts,
	)
	if err != nil {
		return nil, err
	}

	s.lastResult = result
	return result, nil
}

// GetAttemptNumber returns the current attempt number
func (s *PersistentFixSession) GetAttemptNumber() int {
	return s.attemptNumber
}

// GetPreviousAttempts returns all previous attempts
func (s *PersistentFixSession) GetPreviousAttempts() []string {
	return s.previousAttempts
}

// HasMoreAttempts returns true if more fix attempts are possible
func (s *PersistentFixSession) HasMoreAttempts() bool {
	return s.attemptNumber < s.config.MaxAttempts
}

// truncateString truncates a string to max length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// CreateFilesFromResult creates any files specified in the fix result
func (c *Client) CreateFilesFromResult(result *AutoFixResult, baseDir string) error {
	for _, file := range result.FilesToCreate {
		fullPath := filepath.Join(baseDir, file.Path)

		// Ensure directory exists
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Write file
		if err := os.WriteFile(fullPath, []byte(file.Content), 0644); err != nil {
			return fmt.Errorf("failed to create file %s: %w", fullPath, err)
		}
	}
	return nil
}

// getDirectoryContext returns a list of files in the directory
func (c *Client) getDirectoryContext(dir string) string {
	if dir == "" {
		return "Working directory not specified"
	}

	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Skip hidden files and common ignore directories
		if strings.HasPrefix(relPath, ".") ||
			strings.Contains(relPath, "node_modules") ||
			strings.Contains(relPath, "vendor") ||
			strings.Contains(relPath, "target") ||
			strings.Contains(relPath, "dist") ||
			strings.Contains(relPath, "build") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Add file indicator
		prefix := "📄 "
		if info.IsDir() {
			prefix = "📂 "
		}

		files = append(files, prefix+relPath)

		// Limit to 50 files to avoid token limits
		if len(files) >= 50 {
			return fmt.Errorf("limit reached")
		}

		return nil
	})

	if err != nil && err.Error() != "limit reached" {
		return fmt.Sprintf("Error listing files: %v", err)
	}

	if len(files) == 0 {
		return "Directory is empty"
	}

	if len(files) >= 50 {
		files = append(files, "... (truncated)")
	}

	return strings.Join(files, "\n")
}
