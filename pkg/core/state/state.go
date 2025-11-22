package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/glive/core/types"
)

// Manager manages project state and operations
type Manager struct {
	stateDir string
	projects map[string]*types.Project
	mu       sync.RWMutex
}

// Operation represents a reversible operation
type Operation struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Type        string    `json:"type"` // clone, install, file_create, file_modify, command
	Description string    `json:"description"`
	Data        string    `json:"data"` // JSON encoded operation-specific data
	Timestamp   time.Time `json:"timestamp"`
	Reversible  bool      `json:"reversible"`
}

// ProjectState represents the persisted state of a project
type ProjectState struct {
	Project    *types.Project `json:"project"`
	Operations []Operation   `json:"operations"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

// New creates a new state manager
func New(stateDir string) (*Manager, error) {
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	manager := &Manager{
		stateDir: stateDir,
		projects: make(map[string]*types.Project),
	}

	// Load existing projects
	if err := manager.loadProjects(); err != nil {
		return nil, fmt.Errorf("failed to load projects: %w", err)
	}

	return manager, nil
}

// SaveProject saves project state to disk
func (m *Manager) SaveProject(project *types.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	project.UpdatedAt = time.Now()
	m.projects[project.ID] = project

	state := ProjectState{
		Project:    project,
		Operations: []Operation{},
		UpdatedAt:  time.Now(),
	}

	filePath := m.getProjectFilePath(project.ID)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project state: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write project state: %w", err)
	}

	return nil
}

// LoadProject loads a project from disk
func (m *Manager) LoadProject(projectID string) (*types.Project, error) {
	m.mu.RLock()
	if project, exists := m.projects[projectID]; exists {
		m.mu.RUnlock()
		return project, nil
	}
	m.mu.RUnlock()

	filePath := m.getProjectFilePath(projectID)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read project state: %w", err)
	}

	var state ProjectState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project state: %w", err)
	}

	m.mu.Lock()
	m.projects[projectID] = state.Project
	m.mu.Unlock()

	return state.Project, nil
}

// ListProjects returns all projects
func (m *Manager) ListProjects() []*types.Project {
	m.mu.RLock()
	defer m.mu.RUnlock()

	projects := make([]*types.Project, 0, len(m.projects))
	for _, project := range m.projects {
		projects = append(projects, project)
	}

	return projects
}

// DeleteProject removes a project from state
func (m *Manager) DeleteProject(projectID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.projects, projectID)

	filePath := m.getProjectFilePath(projectID)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete project state: %w", err)
	}

	return nil
}

// RecordOperation records an operation for potential rollback
func (m *Manager) RecordOperation(projectID string, op Operation) error {
	filePath := m.getProjectFilePath(projectID)

	// Load current state
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read project state: %w", err)
	}

	var state ProjectState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal project state: %w", err)
	}

	// Add operation
	op.Timestamp = time.Now()
	state.Operations = append(state.Operations, op)
	state.UpdatedAt = time.Now()

	// Save updated state
	data, err = json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project state: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write project state: %w", err)
	}

	return nil
}

// GetOperations returns all operations for a project
func (m *Manager) GetOperations(projectID string) ([]Operation, error) {
	filePath := m.getProjectFilePath(projectID)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read project state: %w", err)
	}

	var state ProjectState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project state: %w", err)
	}

	return state.Operations, nil
}

// Rollback reverts operations for a project
func (m *Manager) Rollback(projectID string) error {
	operations, err := m.GetOperations(projectID)
	if err != nil {
		return fmt.Errorf("failed to get operations: %w", err)
	}

	// Reverse operations in reverse order
	for i := len(operations) - 1; i >= 0; i-- {
		op := operations[i]
		if !op.Reversible {
			continue
		}

		// TODO: Implement actual rollback logic based on operation type
		switch op.Type {
		case "clone":
			// Remove cloned directory
		case "install":
			// Uninstall dependencies (difficult, may skip)
		case "file_create":
			// Delete created file
		case "file_modify":
			// Restore original file
		}
	}

	return nil
}

// loadProjects loads all projects from disk
func (m *Manager) loadProjects() error {
	files, err := os.ReadDir(m.stateDir)
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

		filePath := filepath.Join(m.stateDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var state ProjectState
		if err := json.Unmarshal(data, &state); err != nil {
			continue
		}

		m.projects[state.Project.ID] = state.Project
	}

	return nil
}

// getProjectFilePath returns the file path for a project's state
func (m *Manager) getProjectFilePath(projectID string) string {
	return filepath.Join(m.stateDir, projectID+".json")
}

// UpdateProjectStatus updates a project's status
func (m *Manager) UpdateProjectStatus(projectID string, status types.ProjectStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	project, exists := m.projects[projectID]
	if !exists {
		return fmt.Errorf("project not found: %s", projectID)
	}

	project.Status = status
	project.UpdatedAt = time.Now()

	return m.SaveProject(project)
}
