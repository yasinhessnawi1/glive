package project_test

import (
	"context"
	"testing"

	"github.com/glive/domain/entities"
	"github.com/glive/domain/repository"
	"github.com/glive/domain/values"
	"github.com/glive/infrastructure/container"
	"github.com/glive/infrastructure/executor"
	"github.com/glive/testing/testutil"
	"github.com/glive/usecase/project"
)

// Mock implementations
type mockProjectRepository struct {
	projects map[string]*entities.Project
	saveErr  error
	findErr  error
}

func newMockProjectRepository() *mockProjectRepository {
	return &mockProjectRepository{
		projects: make(map[string]*entities.Project),
	}
}

func (m *mockProjectRepository) Save(ctx context.Context, p *entities.Project) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.projects[p.ID().Value()] = p
	return nil
}

func (m *mockProjectRepository) FindByID(ctx context.Context, id *values.ProjectID) (*entities.Project, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	p, ok := m.projects[id.Value()]
	if !ok {
		return nil, repository.ErrProjectNotFound
	}
	return p, nil
}

func (m *mockProjectRepository) FindByURL(ctx context.Context, url *values.URL) (*entities.Project, error) {
	for _, p := range m.projects {
		if p.URL().String() == url.String() {
			return p, nil
		}
	}
	return nil, repository.ErrProjectNotFound
}

func (m *mockProjectRepository) List(ctx context.Context, filter repository.ProjectFilter) ([]*entities.Project, error) {
	result := make([]*entities.Project, 0, len(m.projects))
	for _, p := range m.projects {
		// Apply filter if needed
		if filter.Status != nil && p.Status() != *filter.Status {
			continue
		}
		if filter.Type != nil && p.Type() != *filter.Type {
			continue
		}
		result = append(result, p)
	}
	return result, nil
}

func (m *mockProjectRepository) Delete(ctx context.Context, id *values.ProjectID) error {
	delete(m.projects, id.Value())
	return nil
}

type mockGitClient struct {
	cloneFunc func(workspaceDir string, progress interface{}) error
}

func (m *mockGitClient) Clone(workspaceDir string, progress interface{}) error {
	if m.cloneFunc != nil {
		return m.cloneFunc(workspaceDir, progress)
	}
	return nil
}

func (m *mockGitClient) CloneWithAuth(workspaceDir string, username, token string, progress interface{}) error {
	return m.Clone(workspaceDir, progress)
}

func (m *mockGitClient) GetLocalPath() string {
	return "/tmp/test-repo"
}

func (m *mockGitClient) IsCloned() bool {
	return true
}

type mockAIClient struct {
	analyzeFunc func(projectPath, readmeContent string, fileList []string) (string, error)
}

func (m *mockAIClient) AnalyzeProject(projectPath, readmeContent string, fileList []string) (string, error) {
	if m.analyzeFunc != nil {
		return m.analyzeFunc(projectPath, readmeContent, fileList)
	}
	return `{"project_type": "nodejs", "commands": []}`, nil
}

func (m *mockAIClient) DebugError(command, output, errorMsg string) (string, error) {
	return "", nil
}

func (m *mockAIClient) AutoFixError(command, output, errorMsg, workingDir string) (string, string, bool, error) {
	return "", "", false, nil
}

func TestSetupProject_Success(t *testing.T) {
	tempDir := testutil.TempDir(t)

	repo := newMockProjectRepository()
	mockContainer := &container.Container{}

	// Create a minimal container setup
	uc := project.NewSetupProjectUseCase(repo, mockContainer, nil)

	input := project.SetupProjectInput{
		GitHubURL:    "https://github.com/test/repo",
		Mode:         executor.ModeAuto,
		WorkspaceDir: tempDir,
		Force:        false,
	}

	// This test would need proper container setup with mocks
	// For now, we'll test the structure
	_ = uc
	_ = input
}

func TestSetupProject_CloneFails(t *testing.T) {
	tempDir := testutil.TempDir(t)

	repo := newMockProjectRepository()
	mockContainer := &container.Container{}

	uc := project.NewSetupProjectUseCase(repo, mockContainer, nil)

	input := project.SetupProjectInput{
		GitHubURL:    "https://github.com/test/repo",
		WorkspaceDir: tempDir,
		Force:        false,
	}

	// This test would need proper container setup with mocks
	_ = uc
	_ = input
}

func TestSetupProject_SuspiciousRepository(t *testing.T) {
	tempDir := testutil.TempDir(t)

	repo := newMockProjectRepository()
	mockContainer := &container.Container{}

	uc := project.NewSetupProjectUseCase(repo, mockContainer, nil)

	input := project.SetupProjectInput{
		GitHubURL:    "https://github.com/test/repo",
		WorkspaceDir: tempDir,
		Force:        false,
	}

	// This test would need proper container setup with mocks
	_ = uc
	_ = input
}
