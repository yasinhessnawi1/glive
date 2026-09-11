package controller

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/glive/infrastructure/container"
	"github.com/glive/infrastructure/executor"
	"github.com/glive/interface/presenter"
	"github.com/glive/usecase/project"
)

// HTTPController handles HTTP requests and delegates to use cases
type HTTPController struct {
	setupUseCase *project.SetupProjectUseCase
	presenter    presenter.JSONPresenter
	container    *container.Container
}

// NewHTTPController creates a new HTTP controller
func NewHTTPController(cont *container.Container) *HTTPController {
	projectRepo := cont.ProjectRepository()
	setupUC := project.NewSetupProjectUseCase(projectRepo, cont, nil)

	return &HTTPController{
		setupUseCase: setupUC,
		presenter:    presenter.NewJSONPresenter(nil),
		container:    cont,
	}
}

// SetupProjectRequest represents the HTTP request for setting up a project
type SetupProjectRequest struct {
	GitHubURL    string `json:"github_url"`
	Mode         string `json:"mode"`
	WorkspaceDir string `json:"workspace_dir"`
}

// SetupProject handles POST /projects endpoint
func (c *HTTPController) SetupProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SetupProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Parse execution mode
	mode := executor.ModeAuto
	switch req.Mode {
	case "assisted":
		mode = executor.ModeAssisted
	case "manual":
		mode = executor.ModeManual
	}

	// Get workspace directory from config if not provided
	workspaceDir := req.WorkspaceDir
	if workspaceDir == "" {
		workspaceDir = c.container.Config().WorkspaceDir
	}

	input := project.SetupProjectInput{
		GitHubURL:    req.GitHubURL,
		Mode:         mode,
		WorkspaceDir: workspaceDir,
	}

	output, err := c.setupUseCase.Execute(r.Context(), input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		c.presenter.PresentError(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	c.presenter.PresentSuccess(output)
}
