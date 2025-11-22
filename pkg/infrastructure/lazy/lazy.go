package lazy

import (
	"sync"

	"github.com/glive/domain/errors"
	"github.com/glive/infrastructure/ai"
)

// Value lazily initializes a value
type Value[T any] struct {
	once  sync.Once
	value T
	err   error
	init  func() (T, error)
}

// New creates a new lazy value
func New[T any](init func() (T, error)) *Value[T] {
	return &Value[T]{init: init}
}

// Get retrieves the value, initializing it if necessary
func (l *Value[T]) Get() (T, error) {
	l.once.Do(func() {
		l.value, l.err = l.init()
	})
	return l.value, l.err
}

// AIClientConfig holds configuration for creating an AI client
type AIClientConfig struct {
	APIKey      string
	APIProvider string
	APIEndpoint string
}

// AIClient interface matches container.AIClient to avoid circular imports
type AIClient interface {
	AnalyzeProject(projectPath string, readmeContent string, fileList []string) (string, error)
	DebugError(command string, output string, errorMsg string) (string, error)
	AutoFixError(command string, output string, errorMsg string, workingDir string) (string, string, bool, error)
}

// LazyAIClient only initializes AI client when needed
type LazyAIClient struct {
	client *Value[AIClient]
}

// NewLazyAIClient creates a new lazy AI client
func NewLazyAIClient(config AIClientConfig) *LazyAIClient {
	return &LazyAIClient{
		client: New(func() (AIClient, error) {
			if config.APIKey == "" {
				return nil, errors.ErrMissingAPIKey
			}
			return ai.NewClient(config.APIKey, config.APIProvider, config.APIEndpoint), nil
		}),
	}
}

// AnalyzeProject analyzes a project using the AI client
func (l *LazyAIClient) AnalyzeProject(projectPath string, readmeContent string, fileList []string) (string, error) {
	client, err := l.client.Get()
	if err != nil {
		return "", err
	}
	return client.AnalyzeProject(projectPath, readmeContent, fileList)
}

// DebugError uses AI to debug an error
func (l *LazyAIClient) DebugError(command string, output string, errorMsg string) (string, error) {
	client, err := l.client.Get()
	if err != nil {
		return "", err
	}
	return client.DebugError(command, output, errorMsg)
}

// AutoFixError attempts to automatically fix a command error
func (l *LazyAIClient) AutoFixError(command string, output string, errorMsg string, workingDir string) (string, string, bool, error) {
	client, err := l.client.Get()
	if err != nil {
		return "", "", false, err
	}
	return client.AutoFixError(command, output, errorMsg, workingDir)
}

