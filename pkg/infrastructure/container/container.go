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

// An Option overrides one of the Container's collaborators after the defaults
// have been wired.
//
// The Container is the composition root: it is the one place concrete types are
// supposed to meet. Until this seam existed it was also a sealed box - every
// field unexported, one constructor, and that constructor hard-wiring a git
// client that clones over the network and an AI client that calls a provider.
// Nothing outside this package could substitute either, so SetupProjectUseCase,
// which reaches all of its collaborators through the Container, could not be
// exercised at all without network and AI access.
//
// Options are additive and apply last, so NewContainer(cfg, logger) behaves
// exactly as before and every existing call site compiles unchanged.
//
// This is a seam, not the fix. The use case depending on a concrete
// *Container rather than on consumer-side interfaces still inverts the
// dependency rule in the standards; GL0 T6 owns that refactor.
type Option func(*Container)

// WithGitClientFactory substitutes the factory that creates Git clients.
func WithGitClientFactory(f GitClientFactory) Option {
	return func(c *Container) { c.gitClientFactory = f }
}

// WithAIClientFactory substitutes the factory that creates AI clients.
func WithAIClientFactory(f AIClientFactory) Option {
	return func(c *Container) { c.aiClientFactory = f }
}

// WithExecutorFactory substitutes the factory that creates command executors.
func WithExecutorFactory(f ExecutorFactory) Option {
	return func(c *Container) { c.executorFactory = f }
}

// WithAnalyzerFactory substitutes the factory that creates project analyzers.
func WithAnalyzerFactory(f AnalyzerFactory) Option {
	return func(c *Container) { c.analyzerFactory = f }
}

// WithScannerFactory substitutes the factory that creates security scanners.
func WithScannerFactory(f ScannerFactory) Option {
	return func(c *Container) { c.scannerFactory = f }
}

// WithProjectRepository substitutes the project repository.
//
// Unlike the five factories this one replaces a collaborator NewContainer builds
// eagerly, on disk under <WorkspaceDir>/.glive. Supplying it lets a caller keep
// project state in memory.
func WithProjectRepository(r repository.ProjectRepository) Option {
	return func(c *Container) { c.projectRepo = r }
}

// NewContainer creates a new dependency injection container.
//
// With no options it wires the production implementations, exactly as before.
// Options are applied after those defaults; see Option.
func NewContainer(cfg *Config, logger Logger, opts ...Option) (*Container, error) {
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

	// Overrides last, so a supplied collaborator always wins over the default.
	for _, opt := range opts {
		if opt != nil {
			opt(container)
		}
	}

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
