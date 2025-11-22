package ai

import (
	"context"
	"fmt"
)

// AnalysisInput represents input for project analysis
type AnalysisInput struct {
	ProjectPath  string
	ReadmeContent string
	FileList     []string
}

// AnalysisOutput represents the output from analysis
type AnalysisOutput struct {
	Response string
}

// FixInput represents input for auto-fix
type FixInput struct {
	Command   string
	Output    string
	ErrorMsg  string
	WorkingDir string
}

// FixOutput represents the output from auto-fix
type FixOutput struct {
	FixedCommand string
	Explanation  string
	CanAutoFix   bool
}

// Provider interface for AI providers
type Provider interface {
	Name() string
	Analyze(ctx context.Context, input AnalysisInput) (*AnalysisOutput, error)
	AutoFix(ctx context.Context, input FixInput) (*FixOutput, error)
	HealthCheck(ctx context.Context) error
}

// ProviderRegistry manages multiple providers with primary/fallback support
type ProviderRegistry struct {
	providers map[string]Provider
	primary   string
	fallback  string
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry(primary string, fallback string) *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
		primary:   primary,
		fallback:  fallback,
	}
}

// RegisterProvider registers a provider with the registry
func (r *ProviderRegistry) RegisterProvider(provider Provider) {
	r.providers[provider.Name()] = provider
}

// Analyze attempts analysis with primary provider, falls back to fallback if needed
func (r *ProviderRegistry) Analyze(ctx context.Context, input AnalysisInput) (*AnalysisOutput, error) {
	// Try primary provider
	if primary, ok := r.providers[r.primary]; ok {
		output, err := primary.Analyze(ctx, input)
		if err == nil {
			return output, nil
		}
		// If primary fails and we have a fallback, try it
		if r.fallback != "" && r.fallback != r.primary {
			if fallback, ok := r.providers[r.fallback]; ok {
				return fallback.Analyze(ctx, input)
			}
		}
		return nil, fmt.Errorf("primary provider failed: %w", err)
	}
	return nil, fmt.Errorf("primary provider %s not found", r.primary)
}

// AutoFix attempts auto-fix with primary provider, falls back to fallback if needed
func (r *ProviderRegistry) AutoFix(ctx context.Context, input FixInput) (*FixOutput, error) {
	// Try primary provider
	if primary, ok := r.providers[r.primary]; ok {
		output, err := primary.AutoFix(ctx, input)
		if err == nil {
			return output, nil
		}
		// If primary fails and we have a fallback, try it
		if r.fallback != "" && r.fallback != r.primary {
			if fallback, ok := r.providers[r.fallback]; ok {
				return fallback.AutoFix(ctx, input)
			}
		}
		return nil, fmt.Errorf("primary provider failed: %w", err)
	}
	return nil, fmt.Errorf("primary provider %s not found", r.primary)
}

// HealthCheck checks health of primary provider
func (r *ProviderRegistry) HealthCheck(ctx context.Context) error {
	if primary, ok := r.providers[r.primary]; ok {
		return primary.HealthCheck(ctx)
	}
	return fmt.Errorf("primary provider %s not found", r.primary)
}

