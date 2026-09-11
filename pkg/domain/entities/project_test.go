package entities_test

import (
	"testing"

	"github.com/glive/domain/entities"
	"github.com/glive/domain/values"
	"github.com/glive/testing/testutil"
)

func TestNewProject(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid github https", "https://github.com/user/repo", false},
		{"valid github ssh", "git@github.com:user/repo.git", false},
		{"valid short form", "user/repo", false},
		{"empty url", "", true},
		{"invalid url", "not-a-url", true},
		{"local file url", "file:///etc/passwd", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoURL, err := values.ParseRepoURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error parsing URL: %v", err)
			}

			// Convert RepoURL to URL for NewProject
			url, err := values.NewURL(repoURL.String())
			if err != nil {
				t.Fatalf("failed to create URL: %v", err)
			}

			// Generate project ID
			projectID := values.GenerateProjectID()

			project, err := entities.NewProject(projectID, url, "test-project")
			if err != nil {
				t.Fatalf("failed to create project: %v", err)
			}

			if project.ID().Value() == "" {
				t.Error("project ID should not be empty")
			}
			if project.Status() != entities.StatusPending {
				t.Errorf("expected status Pending, got %v", project.Status())
			}
		})
	}
}

// driveToStatus walks a freshly-created project (StatusPending) to target using
// only legal transitions, so tests exercise the state machine instead of
// bypassing it. It returns the first rejected transition's error.
func driveToStatus(p *entities.Project, target entities.ProjectStatus) error {
	// Legal routes from StatusPending, per Project.canTransitionTo.
	routes := map[entities.ProjectStatus][]entities.ProjectStatus{
		entities.StatusPending:    {},
		entities.StatusCloning:    {entities.StatusCloning},
		entities.StatusAnalyzing:  {entities.StatusAnalyzing},
		entities.StatusInstalling: {entities.StatusAnalyzing, entities.StatusInstalling},
		entities.StatusReady:      {entities.StatusAnalyzing, entities.StatusInstalling, entities.StatusReady},
		entities.StatusRunning:    {entities.StatusAnalyzing, entities.StatusInstalling, entities.StatusReady, entities.StatusRunning},
		entities.StatusFailed:     {entities.StatusFailed},
		entities.StatusStopped:    {entities.StatusStopped},
	}
	for _, step := range routes[target] {
		if err := p.SetStatus(step); err != nil {
			return err
		}
	}
	return nil
}

func TestProject_CanExecute(t *testing.T) {
	tests := []struct {
		name       string
		status     entities.ProjectStatus
		analysis   *entities.Analysis
		suspicious bool
		canExecute bool
	}{
		{"pending project", entities.StatusPending, nil, false, false},
		{"ready clean", entities.StatusReady, entities.NewAnalysis(entities.ProjectTypeNodeJS), false, true},
		{"ready suspicious", entities.StatusReady, entities.NewAnalysis(entities.ProjectTypeNodeJS), true, false},
		{"cloning", entities.StatusCloning, nil, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, _ := values.NewURL("https://github.com/test/repo")
			projectID := values.GenerateProjectID()
			project, _ := entities.NewProject(projectID, url, "test-project")

			// Drive the project to the target status through the real state machine.
			// SetStatus rejects an illegal transition and returns an error. That error
			// used to be discarded here, so "ready clean" silently stayed Pending
			// (Pending -> Ready is not a legal edge) and CanExecute reported false for
			// a reason the test could not surface.
			if err := driveToStatus(project, tt.status); err != nil {
				t.Fatalf("could not reach status %v: %v", tt.status, err)
			}

			// Set analysis if provided
			if tt.analysis != nil {
				if tt.suspicious {
					tt.analysis.SetSuspicious([]string{"test reason"})
				}
				project.SetAnalysis(tt.analysis)
			}

			if got := project.CanExecute(); got != tt.canExecute {
				t.Errorf("CanExecute() = %v, want %v", got, tt.canExecute)
			}
		})
	}
}

func TestProject_StatusTransitions(t *testing.T) {
	url, _ := values.NewURL("https://github.com/test/repo")
	projectID := values.GenerateProjectID()
	project, err := entities.NewProject(projectID, url, "test-project")
	testutil.AssertNoError(t, err)

	// Initial state should be Pending
	if project.Status() != entities.StatusPending {
		t.Errorf("expected initial status Pending, got %v", project.Status())
	}

	// Transition to Cloning
	err = project.SetStatus(entities.StatusCloning)
	testutil.AssertNoError(t, err)
	if project.Status() != entities.StatusCloning {
		t.Errorf("expected status Cloning, got %v", project.Status())
	}

	// Transition to Analyzing
	err = project.SetStatus(entities.StatusAnalyzing)
	testutil.AssertNoError(t, err)
	if project.Status() != entities.StatusAnalyzing {
		t.Errorf("expected status Analyzing, got %v", project.Status())
	}
}

func TestProject_MarkSuspicious(t *testing.T) {
	url, _ := values.NewURL("https://github.com/test/repo")
	projectID := values.GenerateProjectID()
	project, err := entities.NewProject(projectID, url, "test-project")
	testutil.AssertNoError(t, err)

	err = project.SetStatus(entities.StatusAnalyzing)
	testutil.AssertNoError(t, err)

	analysis := entities.NewAnalysis(entities.ProjectTypeNodeJS)
	reasons := []string{"contains eval()", "suspicious pattern"}
	analysis.SetSuspicious(reasons)
	project.SetAnalysis(analysis)

	if !project.Analysis().IsSuspicious() {
		t.Error("expected project to be marked as suspicious")
	}

	if len(project.Analysis().SuspiciousReasons()) != len(reasons) {
		t.Errorf("expected %d suspicious reasons, got %d", len(reasons), len(project.Analysis().SuspiciousReasons()))
	}
}

func TestProject_SetLocalPath(t *testing.T) {
	url, _ := values.NewURL("https://github.com/test/repo")
	projectID := values.GenerateProjectID()
	project, err := entities.NewProject(projectID, url, "test-project")
	testutil.AssertNoError(t, err)

	path, err := values.NewPath("/tmp/test-project")
	testutil.AssertNoError(t, err)

	err = project.SetLocalPath(path)
	testutil.AssertNoError(t, err)

	if project.LocalPath() == nil {
		t.Error("expected local path to be set")
	}

	if project.LocalPath().Value() != "/tmp/test-project" {
		t.Errorf("expected local path /tmp/test-project, got %s", project.LocalPath().Value())
	}
}
