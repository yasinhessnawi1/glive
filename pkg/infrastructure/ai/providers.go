package ai

import (
	"context"
	"fmt"
	"net/http"

	securehttp "github.com/glive/infrastructure/http"
)

// BaseHTTPProvider provides common functionality for HTTP-based providers
type BaseHTTPProvider struct {
	name      string
	apiKey    string
	endpoint  string
	model     string
	httpClient *http.Client
}

// NewBaseHTTPProvider creates a new base HTTP provider
func NewBaseHTTPProvider(name, apiKey, endpoint, model string) *BaseHTTPProvider {
	return &BaseHTTPProvider{
		name:      name,
		apiKey:    apiKey,
		endpoint:  endpoint,
		model:     model,
		httpClient: securehttp.OptimizedClient(),
	}
}

// Name returns the provider name
func (p *BaseHTTPProvider) Name() string {
	return p.name
}

// HealthCheck performs a simple health check
func (p *BaseHTTPProvider) HealthCheck(ctx context.Context) error {
	// Simple health check - try a minimal request
	// For now, just check if we have an API key
	if p.apiKey == "" {
		return fmt.Errorf("no API key configured for %s", p.name)
	}
	return nil
}

// DeepSeekProvider implements DeepSeek API
type DeepSeekProvider struct {
	*BaseHTTPProvider
	client *Client
}

// NewDeepSeekProvider creates a new DeepSeek provider
func NewDeepSeekProvider(apiKey, endpoint string) *DeepSeekProvider {
	if endpoint == "" {
		endpoint = "https://api.deepseek.com/v1/chat/completions"
	}
	base := NewBaseHTTPProvider("deepseek", apiKey, endpoint, "deepseek-chat")
	return &DeepSeekProvider{
		BaseHTTPProvider: base,
		client:           NewClient(apiKey, "deepseek", endpoint),
	}
}

// Analyze performs project analysis
func (p *DeepSeekProvider) Analyze(ctx context.Context, input AnalysisInput) (*AnalysisOutput, error) {
	response, err := p.client.AnalyzeProject(input.ProjectPath, input.ReadmeContent, input.FileList)
	if err != nil {
		return nil, err
	}
	return &AnalysisOutput{Response: response}, nil
}

// AutoFix attempts to auto-fix an error
func (p *DeepSeekProvider) AutoFix(ctx context.Context, input FixInput) (*FixOutput, error) {
	fixedCmd, explanation, canFix, err := p.client.AutoFixError(input.Command, input.Output, input.ErrorMsg, input.WorkingDir)
	if err != nil {
		return nil, err
	}
	return &FixOutput{
		FixedCommand: fixedCmd,
		Explanation:  explanation,
		CanAutoFix:   canFix,
	}, nil
}

// OpenAIProvider implements OpenAI API
type OpenAIProvider struct {
	*BaseHTTPProvider
	client *Client
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey, endpoint string, model string) *OpenAIProvider {
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1/chat/completions"
	}
	if model == "" {
		model = "gpt-4"
	}
	base := NewBaseHTTPProvider("openai", apiKey, endpoint, model)
	return &OpenAIProvider{
		BaseHTTPProvider: base,
		client:           NewClient(apiKey, "openai", endpoint),
	}
}

// Analyze performs project analysis
func (p *OpenAIProvider) Analyze(ctx context.Context, input AnalysisInput) (*AnalysisOutput, error) {
	response, err := p.client.AnalyzeProject(input.ProjectPath, input.ReadmeContent, input.FileList)
	if err != nil {
		return nil, err
	}
	return &AnalysisOutput{Response: response}, nil
}

// AutoFix attempts to auto-fix an error
func (p *OpenAIProvider) AutoFix(ctx context.Context, input FixInput) (*FixOutput, error) {
	fixedCmd, explanation, canFix, err := p.client.AutoFixError(input.Command, input.Output, input.ErrorMsg, input.WorkingDir)
	if err != nil {
		return nil, err
	}
	return &FixOutput{
		FixedCommand: fixedCmd,
		Explanation:  explanation,
		CanAutoFix:   canFix,
	}, nil
}

// AnthropicProvider implements Anthropic Claude API
type AnthropicProvider struct {
	*BaseHTTPProvider
	client *Client
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(apiKey, endpoint string, model string) *AnthropicProvider {
	if endpoint == "" {
		endpoint = "https://api.anthropic.com/v1/messages"
	}
	if model == "" {
		model = "claude-3-sonnet-20240229"
	}
	base := NewBaseHTTPProvider("anthropic", apiKey, endpoint, model)
	return &AnthropicProvider{
		BaseHTTPProvider: base,
		client:           NewClient(apiKey, "claude", endpoint),
	}
}

// Analyze performs project analysis
func (p *AnthropicProvider) Analyze(ctx context.Context, input AnalysisInput) (*AnalysisOutput, error) {
	response, err := p.client.AnalyzeProject(input.ProjectPath, input.ReadmeContent, input.FileList)
	if err != nil {
		return nil, err
	}
	return &AnalysisOutput{Response: response}, nil
}

// AutoFix attempts to auto-fix an error
func (p *AnthropicProvider) AutoFix(ctx context.Context, input FixInput) (*FixOutput, error) {
	fixedCmd, explanation, canFix, err := p.client.AutoFixError(input.Command, input.Output, input.ErrorMsg, input.WorkingDir)
	if err != nil {
		return nil, err
	}
	return &FixOutput{
		FixedCommand: fixedCmd,
		Explanation:  explanation,
		CanAutoFix:   canFix,
	}, nil
}

