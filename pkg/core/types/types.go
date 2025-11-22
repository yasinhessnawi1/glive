package types

import "time"

// ExecutionMode defines how GLive should run
type ExecutionMode string

const (
	ModeAuto     ExecutionMode = "auto"     // Fully automatic
	ModeAssisted ExecutionMode = "assisted" // User approval for each action
	ModeManual   ExecutionMode = "manual"   // Instructions only
)

// ProjectType represents the detected project type
type ProjectType string

const (
	ProjectTypeNodeJS   ProjectType = "nodejs"
	ProjectTypePython   ProjectType = "python"
	ProjectTypeGo       ProjectType = "go"
	ProjectTypeRust     ProjectType = "rust"
	ProjectTypeJava     ProjectType = "java"
	ProjectTypeDocker   ProjectType = "docker"
	ProjectTypeUnknown  ProjectType = "unknown"
	ProjectTypePolyglot ProjectType = "polyglot"
)

// Project represents a GitHub project being processed
type Project struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	GitHubURL     string        `json:"github_url"`
	LocalPath     string        `json:"local_path"`
	Type          ProjectType   `json:"type"`
	DetectedTypes []ProjectType `json:"detected_types"`
	Status        ProjectStatus `json:"status"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// ProjectStatus represents the current state of a project
type ProjectStatus string

const (
	StatusPending    ProjectStatus = "pending"
	StatusCloning    ProjectStatus = "cloning"
	StatusAnalyzing  ProjectStatus = "analyzing"
	StatusInstalling ProjectStatus = "installing"
	StatusRunning    ProjectStatus = "running"
	StatusReady      ProjectStatus = "ready"
	StatusFailed     ProjectStatus = "failed"
	StatusStopped    ProjectStatus = "stopped"
)

// AnalysisResult contains the AI analysis of a project
type AnalysisResult struct {
	ProjectType        ProjectType  `json:"project_type"`
	DetectedLanguages  []string     `json:"detected_languages"`
	PackageManagers    []string     `json:"package_managers"`
	EntryPoints        []string     `json:"entry_points"`
	Dependencies       []Dependency `json:"dependencies"`
	SystemRequirements []string     `json:"system_requirements"`
	Commands           []Command    `json:"commands"`
	IsSuspicious       bool         `json:"is_suspicious"`
	SuspiciousReasons  []string     `json:"suspicious_reasons"`
	EstimatedSize      string       `json:"estimated_size"`
	Description        string       `json:"description"`
}

// Dependency represents a required dependency
type Dependency struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Type      string `json:"type"` // system, language, package
	Installed bool   `json:"installed"`
}

// Command represents a command to be executed
type Command struct {
	ID          string            `json:"id"`
	Description string            `json:"description"`
	Command     string            `json:"command"`
	WorkingDir  string            `json:"working_dir"`
	Stage       string            `json:"stage"` // setup, build, run
	Required    bool              `json:"required"`
	Status      CommandStatus     `json:"status"`
	Output      string            `json:"output"`
	Error       string            `json:"error"`
	ExitCode    int               `json:"exit_code"`
	Env         map[string]string `json:"env,omitempty"`
}

// CommandStatus represents the execution status of a command
type CommandStatus string

const (
	CommandPending   CommandStatus = "pending"
	CommandRunning   CommandStatus = "running"
	CommandCompleted CommandStatus = "completed"
	CommandFailed    CommandStatus = "failed"
	CommandSkipped   CommandStatus = "skipped"
)

// ProgressUpdate represents a progress event
type ProgressUpdate struct {
	ProjectID  string    `json:"project_id"`
	Stage      string    `json:"stage"`
	Message    string    `json:"message"`
	Percentage int       `json:"percentage"`
	Timestamp  time.Time `json:"timestamp"`
	CommandID  string    `json:"command_id,omitempty"`
}

// Config represents GLive configuration
type Config struct {
	APIKey        string        `json:"api_key"`
	APIProvider   string        `json:"api_provider"` // deepseek, openai, claude, local
	APIEndpoint   string        `json:"api_endpoint"`
	DefaultMode   ExecutionMode `json:"default_mode"`
	WorkspaceDir  string        `json:"workspace_dir"`
	MaxConcurrent int           `json:"max_concurrent"`
	EnableSandbox bool          `json:"enable_sandbox"`
	AgentPort     int           `json:"agent_port"`

	// AI-First configuration
	AIFirst *AIFirstConfig `json:"ai_first,omitempty"`

	// Sandbox configuration
	Sandbox *SandboxConfig `json:"sandbox,omitempty"`
}

// SandboxConfig contains sandbox execution configuration
type SandboxConfig struct {
	Enabled        bool   `json:"enabled"`
	Root           string `json:"root"`
	NetworkEnabled bool   `json:"network_enabled"`
	CPULimit       string `json:"cpu_limit"`
	MemoryLimit    string `json:"memory_limit"`
	Timeout        string `json:"timeout"`
	CleanupOnExit  bool   `json:"cleanup_on_exit"`
}

// AIFirstConfig contains AI-first execution configuration
type AIFirstConfig struct {
	PrimaryMode         string  `json:"primary_mode"`         // ai_first, traditional, hybrid
	ConfidenceThreshold float64 `json:"confidence_threshold"` // Min confidence to proceed (default: 0.7)
	EnableFallback      bool    `json:"enable_fallback"`      // Fallback to traditional on AI failure
	EnableAIMonitoring  bool    `json:"enable_ai_monitoring"` // Monitor execution with AI
	EnableAutoRecovery  bool    `json:"enable_auto_recovery"` // Enable autonomous recovery
	EnableLearning      bool    `json:"enable_learning"`      // Learn from outcomes
	FallbackStrategy    string  `json:"fallback_strategy"`    // traditional, cached, heuristic
}
