package entities

import (
	"fmt"
	"time"

	"github.com/glive/domain/errors"
	"github.com/glive/domain/values"
)

// Project represents a GitHub project entity
type Project struct {
	id          *values.ProjectID
	url         *values.URL
	name        string
	localPath   *values.Path
	projectType ProjectType
	status      ProjectStatus
	analysis    *Analysis
	createdAt   time.Time
	updatedAt   time.Time
}

// ProjectType represents the type of project
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

// NewProject creates a new Project entity
func NewProject(id *values.ProjectID, url *values.URL, name string) (*Project, error) {
	if id == nil {
		return nil, errors.NewDomainError(errors.ErrCodeInvalidInput, "project ID cannot be nil")
	}
	if url == nil {
		return nil, errors.NewDomainError(errors.ErrCodeInvalidInput, "project URL cannot be nil")
	}
	if name == "" {
		return nil, errors.NewDomainError(errors.ErrCodeInvalidInput, "project name cannot be empty")
	}

	now := time.Now()
	return &Project{
		id:        id,
		url:       url,
		name:      name,
		status:    StatusPending,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// ID returns the project ID
func (p *Project) ID() *values.ProjectID {
	return p.id
}

// URL returns the project URL
func (p *Project) URL() *values.URL {
	return p.url
}

// Name returns the project name
func (p *Project) Name() string {
	return p.name
}

// LocalPath returns the local filesystem path
func (p *Project) LocalPath() *values.Path {
	return p.localPath
}

// Type returns the project type
func (p *Project) Type() ProjectType {
	return p.projectType
}

// Status returns the project status
func (p *Project) Status() ProjectStatus {
	return p.status
}

// Analysis returns the project analysis
func (p *Project) Analysis() *Analysis {
	return p.analysis
}

// CreatedAt returns when the project was created
func (p *Project) CreatedAt() time.Time {
	return p.createdAt
}

// UpdatedAt returns when the project was last updated
func (p *Project) UpdatedAt() time.Time {
	return p.updatedAt
}

// SetLocalPath sets the local filesystem path
func (p *Project) SetLocalPath(path *values.Path) error {
	if path == nil {
		return errors.NewDomainError(errors.ErrCodeInvalidInput, "local path cannot be nil")
	}
	p.localPath = path
	p.updatedAt = time.Now()
	return nil
}

// SetType sets the project type
func (p *Project) SetType(projectType ProjectType) {
	p.projectType = projectType
	p.updatedAt = time.Now()
}

// SetStatus sets the project status
func (p *Project) SetStatus(status ProjectStatus) error {
	if !p.canTransitionTo(status) {
		return errors.NewDomainError(
			errors.ErrCodeInvalidState,
			fmt.Sprintf("cannot transition from %s to %s", p.status, status),
		)
	}
	p.status = status
	p.updatedAt = time.Now()
	return nil
}

// SetAnalysis sets the project analysis
func (p *Project) SetAnalysis(analysis *Analysis) {
	p.analysis = analysis
	p.updatedAt = time.Now()
}

// canTransitionTo checks if a status transition is valid
func (p *Project) canTransitionTo(newStatus ProjectStatus) bool {
	validTransitions := map[ProjectStatus][]ProjectStatus{
		StatusPending:    {StatusCloning, StatusAnalyzing, StatusFailed, StatusStopped},
		StatusCloning:    {StatusAnalyzing, StatusFailed, StatusStopped},
		StatusAnalyzing:  {StatusInstalling, StatusFailed, StatusStopped},
		StatusInstalling: {StatusReady, StatusFailed, StatusStopped},
		StatusRunning:    {StatusReady, StatusStopped, StatusFailed},
		StatusReady:      {StatusRunning, StatusStopped, StatusCloning, StatusAnalyzing}, // Allow re-setup
		StatusFailed:     {StatusPending, StatusCloning, StatusAnalyzing, StatusInstalling}, // Can retry from any step
		StatusStopped:    {StatusRunning, StatusPending, StatusCloning},
	}

	allowed, exists := validTransitions[p.status]
	if !exists {
		return false
	}

	for _, allowedStatus := range allowed {
		if allowedStatus == newStatus {
			return true
		}
	}

	return false
}

// CanExecute checks if the project can be executed
func (p *Project) CanExecute() bool {
	return p.status == StatusReady &&
		p.analysis != nil &&
		!p.analysis.IsSuspicious()
}

// IsReady checks if the project is ready to use
func (p *Project) IsReady() bool {
	return p.status == StatusReady
}

// IsFailed checks if the project has failed
func (p *Project) IsFailed() bool {
	return p.status == StatusFailed
}

// CanRetry checks if the project can be retried
func (p *Project) CanRetry() bool {
	return p.status == StatusFailed
}

// SetTimestamps sets the created and updated timestamps (for loading from persistence)
func (p *Project) SetTimestamps(createdAt, updatedAt time.Time) {
	p.createdAt = createdAt
	p.updatedAt = updatedAt
}

// SetStatusUnsafe sets the project status without validation (for loading from persistence)
func (p *Project) SetStatusUnsafe(status ProjectStatus) {
	p.status = status
	// Don't update updatedAt here - it should be restored from persistence
}

