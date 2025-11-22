package core

// Re-export types from the types subpackage to maintain backward compatibility
import "github.com/glive/core/types"

type (
	ExecutionMode  = types.ExecutionMode
	ProjectType    = types.ProjectType
	Project        = types.Project
	ProjectStatus  = types.ProjectStatus
	AnalysisResult = types.AnalysisResult
	Dependency     = types.Dependency
	Command        = types.Command
	CommandStatus  = types.CommandStatus
	ProgressUpdate = types.ProgressUpdate
	Config         = types.Config
)

// Re-export constants
const (
	ModeAuto     = types.ModeAuto
	ModeAssisted = types.ModeAssisted
	ModeManual   = types.ModeManual

	ProjectTypeNodeJS   = types.ProjectTypeNodeJS
	ProjectTypePython   = types.ProjectTypePython
	ProjectTypeGo       = types.ProjectTypeGo
	ProjectTypeRust     = types.ProjectTypeRust
	ProjectTypeJava     = types.ProjectTypeJava
	ProjectTypeDocker   = types.ProjectTypeDocker
	ProjectTypeUnknown  = types.ProjectTypeUnknown
	ProjectTypePolyglot = types.ProjectTypePolyglot

	StatusPending    = types.StatusPending
	StatusCloning    = types.StatusCloning
	StatusAnalyzing  = types.StatusAnalyzing
	StatusInstalling = types.StatusInstalling
	StatusRunning    = types.StatusRunning
	StatusReady      = types.StatusReady
	StatusFailed     = types.StatusFailed
	StatusStopped    = types.StatusStopped

	CommandPending   = types.CommandPending
	CommandRunning   = types.CommandRunning
	CommandCompleted = types.CommandCompleted
	CommandFailed    = types.CommandFailed
	CommandSkipped   = types.CommandSkipped
)
