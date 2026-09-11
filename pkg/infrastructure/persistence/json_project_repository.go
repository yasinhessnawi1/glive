package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/glive/domain/entities"
	"github.com/glive/domain/repository"
	"github.com/glive/domain/values"
	"github.com/glive/infrastructure/filesystem"
)

// JSONProjectRepository implements project persistence using JSON files
type JSONProjectRepository struct {
	stateDir string
	projects map[string]*entities.Project
	safeFS   *filesystem.SafeFileSystem
	mu       sync.RWMutex
}

// projectDTO is the data transfer object for JSON serialization
type projectDTO struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	GitHubURL string       `json:"github_url"`
	LocalPath string       `json:"local_path"`
	Type      string       `json:"type"`
	Status    string       `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	Analysis  *analysisDTO `json:"analysis,omitempty"`
}

type analysisDTO struct {
	ProjectType        string          `json:"project_type"`
	DetectedLanguages  []string        `json:"detected_languages"`
	PackageManagers    []string        `json:"package_managers"`
	EntryPoints        []string        `json:"entry_points"`
	Dependencies       []dependencyDTO `json:"dependencies"`
	SystemRequirements []string        `json:"system_requirements"`
	IsSuspicious       bool            `json:"is_suspicious"`
	SuspiciousReasons  []string        `json:"suspicious_reasons"`
	EstimatedSize      string          `json:"estimated_size"`
	Description        string          `json:"description"`
}

type dependencyDTO struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Type      string `json:"type"`
	Installed bool   `json:"installed"`
}

// ProjectState represents the persisted state of a project
type ProjectState struct {
	Project   *projectDTO `json:"project"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// NewJSONProjectRepository creates a new JSON-based project repository
func NewJSONProjectRepository(stateDir string) (repository.ProjectRepository, error) {
	// Create safe file system with state directory as allowed root
	safeFS, err := filesystem.NewSafeFileSystem(stateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create safe file system: %w", err)
	}

	if err := safeFS.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	repo := &JSONProjectRepository{
		stateDir: stateDir,
		projects: make(map[string]*entities.Project),
		safeFS:   safeFS,
	}

	// Load existing projects
	if err := repo.loadProjects(); err != nil {
		return nil, fmt.Errorf("failed to load projects: %w", err)
	}

	return repo, nil
}

// Save saves a project to persistent storage
func (r *JSONProjectRepository) Save(ctx context.Context, project *entities.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.projects[project.ID().Value()] = project

	dto := r.entityToDTO(project)
	state := ProjectState{
		Project:   dto,
		UpdatedAt: time.Now(),
	}

	filePath := r.getProjectFilePath(project.ID().Value())
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project state: %w", err)
	}

	if err := r.safeFS.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write project state: %w", err)
	}

	return nil
}

// FindByID finds a project by its ID
func (r *JSONProjectRepository) FindByID(ctx context.Context, id *values.ProjectID) (*entities.Project, error) {
	r.mu.RLock()
	if project, exists := r.projects[id.Value()]; exists {
		r.mu.RUnlock()
		return project, nil
	}
	r.mu.RUnlock()

	filePath := r.getProjectFilePath(id.Value())
	data, err := r.safeFS.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("project not found: %s", id.Value())
		}
		return nil, fmt.Errorf("failed to read project state: %w", err)
	}

	var state ProjectState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project state: %w", err)
	}

	project, err := r.dtoToEntity(state.Project)
	if err != nil {
		return nil, fmt.Errorf("failed to convert DTO to entity: %w", err)
	}

	r.mu.Lock()
	r.projects[project.ID().Value()] = project
	r.mu.Unlock()

	return project, nil
}

// FindByURL finds a project by its GitHub URL
func (r *JSONProjectRepository) FindByURL(ctx context.Context, url *values.URL) (*entities.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, project := range r.projects {
		if project.URL().Normalize() == url.Normalize() {
			return project, nil
		}
	}

	return nil, fmt.Errorf("project not found with URL: %s", url.String())
}

// List returns all projects, optionally filtered
func (r *JSONProjectRepository) List(ctx context.Context, filter repository.ProjectFilter) ([]*entities.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	projects := make([]*entities.Project, 0, len(r.projects))
	for _, project := range r.projects {
		// Apply filters
		if filter.Status != nil && project.Status() != *filter.Status {
			continue
		}
		if filter.Type != nil && project.Type() != *filter.Type {
			continue
		}
		projects = append(projects, project)
	}

	return projects, nil
}

// Delete removes a project from persistent storage
func (r *JSONProjectRepository) Delete(ctx context.Context, id *values.ProjectID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.projects, id.Value())

	filePath := r.getProjectFilePath(id.Value())
	if err := r.safeFS.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete project state: %w", err)
	}

	return nil
}

// entityToDTO converts a domain entity to DTO for serialization
func (r *JSONProjectRepository) entityToDTO(project *entities.Project) *projectDTO {
	dto := &projectDTO{
		ID:        project.ID().Value(),
		Name:      project.Name(),
		GitHubURL: project.URL().String(),
		Type:      string(project.Type()),
		Status:    string(project.Status()),
		CreatedAt: project.CreatedAt(),
		UpdatedAt: project.UpdatedAt(),
	}

	if project.LocalPath() != nil {
		dto.LocalPath = project.LocalPath().String()
	}

	if project.Analysis() != nil {
		analysis := project.Analysis()
		dto.Analysis = &analysisDTO{
			ProjectType:        string(analysis.ProjectType()),
			DetectedLanguages:  analysis.DetectedLanguages(),
			PackageManagers:    analysis.PackageManagers(),
			EntryPoints:        analysis.EntryPoints(),
			SystemRequirements: analysis.SystemRequirements(),
			IsSuspicious:       analysis.IsSuspicious(),
			SuspiciousReasons:  analysis.SuspiciousReasons(),
			EstimatedSize:      analysis.EstimatedSize(),
			Description:        analysis.Description(),
		}

		// Convert dependencies
		for _, dep := range analysis.Dependencies() {
			dto.Analysis.Dependencies = append(dto.Analysis.Dependencies, dependencyDTO{
				Name:      dep.Name,
				Version:   dep.Version,
				Type:      dep.Type,
				Installed: dep.Installed,
			})
		}
	}

	return dto
}

// dtoToEntity converts a DTO to domain entity
func (r *JSONProjectRepository) dtoToEntity(dto *projectDTO) (*entities.Project, error) {
	// Create URL value object
	url, err := values.NewURL(dto.GitHubURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Create ProjectID value object
	id, err := values.NewProjectID(dto.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	// Create project entity
	project, err := entities.NewProject(id, url, dto.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	// Set local path if present
	if dto.LocalPath != "" {
		localPath, err := values.NewPath(dto.LocalPath)
		if err != nil {
			return nil, fmt.Errorf("invalid local path: %w", err)
		}
		if err := project.SetLocalPath(localPath); err != nil {
			return nil, fmt.Errorf("failed to set local path: %w", err)
		}
	}

	// Set project type
	project.SetType(entities.ProjectType(dto.Type))

	// Set status (use unsafe method to bypass validation when loading from persistence)
	project.SetStatusUnsafe(entities.ProjectStatus(dto.Status))

	// Set analysis if present
	if dto.Analysis != nil {
		analysis := entities.NewAnalysis(entities.ProjectType(dto.Analysis.ProjectType))
		analysis.SetDetectedLanguages(dto.Analysis.DetectedLanguages)
		analysis.SetPackageManagers(dto.Analysis.PackageManagers)
		analysis.SetEntryPoints(dto.Analysis.EntryPoints)
		analysis.SetSystemRequirements(dto.Analysis.SystemRequirements)
		analysis.SetEstimatedSize(dto.Analysis.EstimatedSize)
		analysis.SetDescription(dto.Analysis.Description)

		// Convert dependencies
		dependencies := make([]entities.Dependency, 0, len(dto.Analysis.Dependencies))
		for _, depDTO := range dto.Analysis.Dependencies {
			dependencies = append(dependencies, entities.Dependency{
				Name:      depDTO.Name,
				Version:   depDTO.Version,
				Type:      depDTO.Type,
				Installed: depDTO.Installed,
			})
		}
		analysis.SetDependencies(dependencies)

		if dto.Analysis.IsSuspicious {
			analysis.SetSuspicious(dto.Analysis.SuspiciousReasons)
		}

		project.SetAnalysis(analysis)
	}

	// Restore timestamps from DTO at the end (after all setters that might update timestamps)
	project.SetTimestamps(dto.CreatedAt, dto.UpdatedAt)

	return project, nil
}

// loadProjects loads all projects from disk
func (r *JSONProjectRepository) loadProjects() error {
	files, err := r.safeFS.ReadDir(r.stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(r.stateDir, file.Name())
		data, err := r.safeFS.ReadFile(filePath)
		if err != nil {
			continue
		}

		var state ProjectState
		if err := json.Unmarshal(data, &state); err != nil {
			continue
		}

		project, err := r.dtoToEntity(state.Project)
		if err != nil {
			// Skip invalid projects
			continue
		}

		r.projects[project.ID().Value()] = project
	}

	return nil
}

// getProjectFilePath returns the file path for a project's state
func (r *JSONProjectRepository) getProjectFilePath(projectID string) string {
	return filepath.Join(r.stateDir, projectID+".json")
}
