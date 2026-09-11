package container

import (
	"fmt"
	"path/filepath"

	"github.com/glive/domain/repository"
	"github.com/glive/infrastructure/analyzer"
	"github.com/glive/infrastructure/executor"
	"github.com/glive/infrastructure/git"
	"github.com/glive/infrastructure/lazy"
	"github.com/glive/infrastructure/persistence"
	"github.com/glive/infrastructure/scanner"
)

// Container holds all dependencies and provides access to them
type Container struct {
	config           *Config
	logger           Logger
	projectRepo      repository.ProjectRepository
	gitClientFactory GitClientFactory
	aiClientFactory  AIClientFactory
	executorFactory  ExecutorFactory
	analyzerFactory  AnalyzerFactory
	scannerFactory   ScannerFactory
}

// GitClientFactory creates Git clients for specific repositories
type GitClientFactory interface {
	Create(githubURL string) (GitClient, error)
}

// AIClientFactory creates AI clients
type AIClientFactory interface {
	Create() AIClient
}

// ExecutorFactory creates executors for specific working directories
type ExecutorFactory interface {
	Create(workingDir string, mode executor.ExecutionMode) Executor
}

// AnalyzerFactory creates analyzers for specific project paths
type AnalyzerFactory interface {
	Create(projectPath string) Analyzer
}

// ScannerFactory creates scanners for specific project paths
type ScannerFactory interface {
	Create(projectPath string) Scanner
}

// NewContainer creates a new dependency injection container
func NewContainer(cfg *Config, logger Logger) (*Container, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	// Create project repository
	stateDir := filepath.Join(cfg.WorkspaceDir, ".glive")
	projectRepo, err := persistence.NewJSONProjectRepository(stateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create project repository: %w", err)
	}

	container := &Container{
		config:      cfg,
		logger:      logger,
		projectRepo: projectRepo,
	}

	// Initialize factories
	container.gitClientFactory = &gitClientFactoryImpl{}
	container.aiClientFactory = &aiClientFactoryImpl{
		config: cfg,
	}
	container.executorFactory = &executorFactoryImpl{
		defaultMode:     cfg.DefaultMode,
		aiClientFactory: container.aiClientFactory,
		enableSandbox:   cfg.EnableSandbox,
	}
	container.analyzerFactory = &analyzerFactoryImpl{}
	container.scannerFactory = &scannerFactoryImpl{}

	return container, nil
}

// Config returns the configuration
func (c *Container) Config() *Config {
	return c.config
}

// Logger returns the logger
func (c *Container) Logger() Logger {
	return c.logger
}

// ProjectRepository returns the project repository
func (c *Container) ProjectRepository() repository.ProjectRepository {
	return c.projectRepo
}

// CreateGitClient creates a Git client for a GitHub URL
func (c *Container) CreateGitClient(githubURL string) (GitClient, error) {
	return c.gitClientFactory.Create(githubURL)
}

// CreateAIClient creates an AI client
func (c *Container) CreateAIClient() AIClient {
	return c.aiClientFactory.Create()
}

// CreateExecutor creates an executor for a working directory
func (c *Container) CreateExecutor(workingDir string, mode executor.ExecutionMode) Executor {
	return c.executorFactory.Create(workingDir, mode)
}

// CreateAnalyzer creates an analyzer for a project path
func (c *Container) CreateAnalyzer(projectPath string) Analyzer {
	return c.analyzerFactory.Create(projectPath)
}

// CreateScanner creates a scanner for a project path
func (c *Container) CreateScanner(projectPath string) Scanner {
	return c.scannerFactory.Create(projectPath)
}

// Implementation of factories

type gitClientFactoryImpl struct{}

func (f *gitClientFactoryImpl) Create(githubURL string) (GitClient, error) {
	return git.ParseGitHubURL(githubURL)
}

type aiClientFactoryImpl struct {
	config *Config
	lazy   *lazy.LazyAIClient
}

func (f *aiClientFactoryImpl) Create() AIClient {
	if f.lazy == nil {
		f.lazy = lazy.NewLazyAIClient(lazy.AIClientConfig{
			APIKey:      f.config.APIKey,
			APIProvider: f.config.APIProvider,
			APIEndpoint: f.config.APIEndpoint,
		})
	}
	return f.lazy
}

type executorFactoryImpl struct {
	defaultMode     executor.ExecutionMode
	aiClientFactory AIClientFactory
	enableSandbox   bool
}

func (f *executorFactoryImpl) Create(workingDir string, mode executor.ExecutionMode) Executor {
	if mode == "" {
		mode = f.defaultMode
	}

	// Use sandboxed executor if enabled
	if f.enableSandbox && executor.CheckSandboxAvailable() {
		sandboxConfig := executor.DefaultSandboxConfig()
		return executor.NewSandboxedExecutor(sandboxConfig)
	}

	// Get AI client lazily when needed
	aiClient := f.aiClientFactory.Create()
	return executor.New(workingDir, mode, aiClient)
}

type analyzerFactoryImpl struct{}

func (f *analyzerFactoryImpl) Create(projectPath string) Analyzer {
	return analyzer.New(projectPath)
}

type scannerFactoryImpl struct{}

func (f *scannerFactoryImpl) Create(projectPath string) Scanner {
	return scanner.New(projectPath)
}
