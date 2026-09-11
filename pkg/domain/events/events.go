package events

import (
	"time"

	"github.com/glive/domain/values"
)

// Event represents a domain event
type Event interface {
	OccurredAt() time.Time
	EventType() string
}

// ProjectCloned event is published when a project is cloned
type ProjectCloned struct {
	ProjectID  *values.ProjectID
	URL        string
	LocalPath  string
	occurredAt time.Time
}

func NewProjectCloned(projectID *values.ProjectID, url, localPath string) *ProjectCloned {
	return &ProjectCloned{
		ProjectID:  projectID,
		URL:        url,
		LocalPath:  localPath,
		occurredAt: time.Now(),
	}
}

func (e *ProjectCloned) OccurredAt() time.Time {
	return e.occurredAt
}

func (e *ProjectCloned) EventType() string {
	return "ProjectCloned"
}

// ProjectAnalyzed event is published when a project is analyzed
type ProjectAnalyzed struct {
	ProjectID   *values.ProjectID
	ProjectType string
	occurredAt  time.Time
}

func NewProjectAnalyzed(projectID *values.ProjectID, projectType string) *ProjectAnalyzed {
	return &ProjectAnalyzed{
		ProjectID:   projectID,
		ProjectType: projectType,
		occurredAt:  time.Now(),
	}
}

func (e *ProjectAnalyzed) OccurredAt() time.Time {
	return e.occurredAt
}

func (e *ProjectAnalyzed) EventType() string {
	return "ProjectAnalyzed"
}

// CommandExecuted event is published when a command is executed
type CommandExecuted struct {
	ProjectID  *values.ProjectID
	CommandID  string
	ExitCode   int
	Duration   time.Duration
	Success    bool
	occurredAt time.Time
}

func NewCommandExecuted(projectID *values.ProjectID, commandID string, exitCode int, duration time.Duration, success bool) *CommandExecuted {
	return &CommandExecuted{
		ProjectID:  projectID,
		CommandID:  commandID,
		ExitCode:   exitCode,
		Duration:   duration,
		Success:    success,
		occurredAt: time.Now(),
	}
}

func (e *CommandExecuted) OccurredAt() time.Time {
	return e.occurredAt
}

func (e *CommandExecuted) EventType() string {
	return "CommandExecuted"
}

// ProjectReady event is published when a project is ready
type ProjectReady struct {
	ProjectID  *values.ProjectID
	LocalPath  string
	occurredAt time.Time
}

func NewProjectReady(projectID *values.ProjectID, localPath string) *ProjectReady {
	return &ProjectReady{
		ProjectID:  projectID,
		LocalPath:  localPath,
		occurredAt: time.Now(),
	}
}

func (e *ProjectReady) OccurredAt() time.Time {
	return e.occurredAt
}

func (e *ProjectReady) EventType() string {
	return "ProjectReady"
}

// ProjectFailed event is published when a project setup fails
type ProjectFailed struct {
	ProjectID  *values.ProjectID
	Error      string
	occurredAt time.Time
}

func NewProjectFailed(projectID *values.ProjectID, errorMsg string) *ProjectFailed {
	return &ProjectFailed{
		ProjectID:  projectID,
		Error:      errorMsg,
		occurredAt: time.Now(),
	}
}

func (e *ProjectFailed) OccurredAt() time.Time {
	return e.occurredAt
}

func (e *ProjectFailed) EventType() string {
	return "ProjectFailed"
}
