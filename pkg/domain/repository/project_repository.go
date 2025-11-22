package repository

import (
	"context"

	"github.com/glive/domain/entities"
	"github.com/glive/domain/values"
)

// ProjectRepository defines the interface for project persistence
// This interface belongs to the domain layer and has no knowledge of implementation details
type ProjectRepository interface {
	// Save saves a project to persistent storage
	Save(ctx context.Context, project *entities.Project) error

	// FindByID finds a project by its ID
	FindByID(ctx context.Context, id *values.ProjectID) (*entities.Project, error)

	// FindByURL finds a project by its GitHub URL
	FindByURL(ctx context.Context, url *values.URL) (*entities.Project, error)

	// List returns all projects, optionally filtered
	List(ctx context.Context, filter ProjectFilter) ([]*entities.Project, error)

	// Delete removes a project from persistent storage
	Delete(ctx context.Context, id *values.ProjectID) error
}

// ProjectFilter defines filtering options for listing projects
type ProjectFilter struct {
	Status *entities.ProjectStatus
	Type   *entities.ProjectType
}

