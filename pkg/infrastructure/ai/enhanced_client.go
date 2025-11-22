package ai

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/glive/infrastructure/retry"
	
	prompts "github.com/glive/infrastructure/ai/prompts"
)

// EnhancedClient wraps the provider registry with additional features
type EnhancedClient struct {
	registry      *ProviderRegistry
	promptManager *prompts.PromptManager
	parser        *ResponseParser
	tokenCounter  *TokenCounter
	cache         *AICache
	safetyFilter  *SafetyFilter
	circuitBreaker *CircuitBreaker
	retrier       *retry.Retrier
}

// EnhancedClientConfig configures an enhanced client
type EnhancedClientConfig struct {
	PrimaryProvider   string
	FallbackProvider  string
	APIKey            string
	APIEndpoint       string
	Model             string
	EnableCache       bool
	EnableCircuitBreaker bool
	PromptsPath       string
}

// NewEnhancedClient creates a new enhanced AI client with all features
func NewEnhancedClient(config EnhancedClientConfig) (*EnhancedClient, error) {
	// Create provider registry
	registry := NewProviderRegistry(config.PrimaryProvider, config.FallbackProvider)

	// Register providers based on config
	if config.APIKey != "" {
		switch config.PrimaryProvider {
		case "deepseek":
			registry.RegisterProvider(NewDeepSeekProvider(config.APIKey, config.APIEndpoint))
		case "openai":
			registry.RegisterProvider(NewOpenAIProvider(config.APIKey, config.APIEndpoint, config.Model))
		case "anthropic", "claude":
			registry.RegisterProvider(NewAnthropicProvider(config.APIKey, config.APIEndpoint, config.Model))
		}

		// Register fallback if different
		if config.FallbackProvider != "" && config.FallbackProvider != config.PrimaryProvider {
			switch config.FallbackProvider {
			case "deepseek":
				registry.RegisterProvider(NewDeepSeekProvider(config.APIKey, config.APIEndpoint))
			case "openai":
				registry.RegisterProvider(NewOpenAIProvider(config.APIKey, config.APIEndpoint, config.Model))
			case "anthropic", "claude":
				registry.RegisterProvider(NewAnthropicProvider(config.APIKey, config.APIEndpoint, config.Model))
			}
		}
	}

	// Create prompt manager
	promptsPath := config.PromptsPath
	if promptsPath == "" {
		promptsPath = filepath.Join("pkg", "infrastructure", "ai", "prompts")
	}
	promptManager, err := prompts.NewPromptManager(promptsPath)
	if err != nil {
		// If prompts can't be loaded, create empty manager (will use defaults)
		promptManager, _ = prompts.NewPromptManager("")
	}

	// Create other components
	parser := NewResponseParser()
	tokenCounter := NewTokenCounter()
	safetyFilter := NewSafetyFilter()

	var aiCache *AICache
	if config.EnableCache {
		aiCache = DefaultAICache()
	}

	var circuitBreaker *CircuitBreaker
	if config.EnableCircuitBreaker {
		circuitBreaker = DefaultCircuitBreaker()
	}

	// Create retrier for network operations
	retrier := retry.New(retry.NetworkConfig(), nil)

	return &EnhancedClient{
		registry:       registry,
		promptManager: promptManager,
		parser:        parser,
		tokenCounter:  tokenCounter,
		cache:          aiCache,
		safetyFilter:  safetyFilter,
		circuitBreaker: circuitBreaker,
		retrier:       retrier,
	}, nil
}

// AnalyzeProject analyzes a project using the enhanced client
func (c *EnhancedClient) AnalyzeProject(projectPath string, readmeContent string, fileList []string) (string, error) {
	// Create analysis input
	input := AnalysisInput{
		ProjectPath:   projectPath,
		ReadmeContent: readmeContent,
		FileList:      fileList,
	}

	// Check cache first
	if c.cache != nil {
		if cached, ok := c.cache.Get(input); ok {
			return cached.Response, nil
		}
	}

	// Sanitize input
	input = c.safetyFilter.SanitizeAnalysisInput(input)

	// Truncate content if needed
	input.ReadmeContent = c.tokenCounter.TruncateReadme(input.ReadmeContent)
	input.FileList = c.tokenCounter.TruncateFileList(input.FileList)

	// Optimize content using token optimizer
	if c.tokenCounter != nil {
		optimizer := prompts.NewTokenOptimizer(func(text string) int {
			return c.tokenCounter.Count(text)
		})
		input.ReadmeContent = optimizer.RemoveRedundantWhitespace(input.ReadmeContent)
		// Compress output if present
		if len(input.FileList) > 0 {
			fileListStr := strings.Join(input.FileList, "\n")
			optimized := optimizer.ExtractKeyInformation(fileListStr)
			input.FileList = strings.Split(optimized, "\n")
		}
	}

	// Execute with circuit breaker and retry
	ctx := context.Background()
	var output *AnalysisOutput
	var err error

	executeFn := func() error {
		if c.circuitBreaker != nil {
			err := c.circuitBreaker.Call(func() error {
				var cbErr error
				output, cbErr = c.registry.Analyze(ctx, input)
				return cbErr
			})
			if err != nil {
				return err
			}
		} else {
			output, err = c.registry.Analyze(ctx, input)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// Retry with circuit breaker protection
	if err := c.retrier.Do(ctx, "ai_analyze", executeFn); err != nil {
		return "", fmt.Errorf("AI analysis failed: %w", err)
	}

	// Cache the result
	if c.cache != nil && output != nil {
		c.cache.Set(input, output)
	}

	return output.Response, nil
}

// DebugError debugs an error using AI
func (c *EnhancedClient) DebugError(command string, output string, errorMsg string) (string, error) {
	// Get prompt from manager or use default
	var prompt string
	if c.promptManager != nil {
		vars := map[string]interface{}{
			"command":   command,
			"output":    output,
			"error_msg": errorMsg,
		}
		rendered, err := c.promptManager.Render("debug_error", vars)
		if err == nil {
			prompt = rendered
		}
	}

	// Fallback to default prompt
	if prompt == "" {
		prompt = fmt.Sprintf(`A command failed during project setup. Help debug and suggest a fix.

Command: %s
Output: %s
Error: %s

Provide a clear explanation of the error and specific steps to fix it.`, command, output, errorMsg)
	}

	// For now, use the registry's primary provider directly
	// In a full implementation, we'd add a Debug method to the Provider interface
	// This would require extending the Provider interface
	
	return "", fmt.Errorf("debug error not yet implemented with enhanced client - use legacy client for now")
}

// AutoFixError attempts to auto-fix an error
func (c *EnhancedClient) AutoFixError(command string, output string, errorMsg string, workingDir string) (string, string, bool, error) {
	input := FixInput{
		Command:    command,
		Output:     output,
		ErrorMsg:   errorMsg,
		WorkingDir: workingDir,
	}

	ctx := context.Background()
	var fixOutput *FixOutput
	var err error

	executeFn := func() error {
		if c.circuitBreaker != nil {
			err := c.circuitBreaker.Call(func() error {
				var cbErr error
				fixOutput, cbErr = c.registry.AutoFix(ctx, input)
				return cbErr
			})
			if err != nil {
				return err
			}
		} else {
			fixOutput, err = c.registry.AutoFix(ctx, input)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// Retry with circuit breaker protection
	if err := c.retrier.Do(ctx, "ai_autofix", executeFn); err != nil {
		return "", "", false, fmt.Errorf("AI auto-fix failed: %w", err)
	}

	return fixOutput.FixedCommand, fixOutput.Explanation, fixOutput.CanAutoFix, nil
}

