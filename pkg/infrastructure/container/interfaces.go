package container

import (
	"context"
	"io"

	"github.com/glive/domain/repository"
	"github.com/glive/infrastructure/analyzer"
	"github.com/glive/infrastructure/executor"
	"github.com/glive/infrastructure/scanner"
)

// GitClient interface for Git operations
type GitClient interface {
	Clone(workspaceDir string, progress io.Writer) error
	CloneWithAuth(workspaceDir string, username, token string, progress io.Writer) error
	GetLocalPath() string
	IsCloned() bool
	DeleteExisting() error
	RenameExisting() (string, error)
}

// AIClient interface for AI operations
type AIClient interface {
	AnalyzeProject(projectPath string, readmeContent string, fileList []string) (string, error)
	DebugError(command string, output string, errorMsg string) (string, error)
	AutoFixError(command string, output string, errorMsg string, workingDir string) (string, string, bool, error)
}

// Executor interface for command execution
// All long-running operations MUST accept context for cancellation
type Executor interface {
	Execute(ctx context.Context, cmd *executor.Command, outputHandler executor.OutputHandler) error
	ExecuteMultiple(ctx context.Context, commands []*executor.Command, outputHandler executor.OutputHandler) error
}

// Analyzer interface for project analysis
type Analyzer interface {
	Analyze() (*analyzer.AnalysisResult, error)
}

// Scanner interface for security scanning
type Scanner interface {
	Scan() (*scanner.ScanResult, error)
}

// ProjectRepository interface (re-export from domain)
type ProjectRepository = repository.ProjectRepository

// Logger interface for logging
type Logger interface {
	Info(msg string)
	Error(msg string, err error)
	Debug(msg string)
}

// Config represents application configuration
type Config struct {
	APIKey        string
	APIProvider   string
	APIEndpoint   string
	WorkspaceDir  string
	DefaultMode   executor.ExecutionMode
	MaxConcurrent int
	EnableSandbox bool
	AgentPort     int
	
	// Observability configuration
	LogLevel      string // DEBUG, INFO, WARN, ERROR
	LogFormat     string // json, text
	LogOutputPath string // Path to log file (empty for stdout)
	MetricsEnabled bool
	TracingEnabled bool
	AuditLogPath   string // Path to audit log file
	LogMaxSize     int64  // Max log file size in bytes
	LogMaxAge      int    // Max log file age in days
	LogMaxBackups  int    // Max number of backup log files
}

