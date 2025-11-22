package tui

import (
	"context"
	"io"

	"github.com/glive/infrastructure/container"
	"github.com/glive/infrastructure/executor"
	"github.com/glive/usecase/project"
)

// TUIController bridges TUI with use cases
type TUIController struct {
	container   *container.Container
	setupUseCase *project.SetupProjectUseCase
	output      io.Writer
}

// NewTUIController creates a new TUI controller
func NewTUIController(cont *container.Container, output io.Writer) *TUIController {
	if output == nil {
		output = io.Discard // TUI doesn't use stdout for output
	}

	projectRepo := cont.ProjectRepository()
	setupUC := project.NewSetupProjectUseCase(projectRepo, cont, output)

	return &TUIController{
		container:   cont,
		setupUseCase: setupUC,
		output:      output,
	}
}

// SetProgressCallback sets the progress callback for the setup use case
func (c *TUIController) SetProgressCallback(callback project.ProgressCallback) {
	c.setupUseCase.SetProgressCallback(callback)
}

// RunProject runs a project from a GitHub URL
func (c *TUIController) RunProject(ctx context.Context, githubURL string, mode executor.ExecutionMode, workspaceDir string, force bool) error {
	input := project.SetupProjectInput{
		GitHubURL:    githubURL,
		Mode:         mode,
		WorkspaceDir: workspaceDir,
		Force:        force,
	}

	_, err := c.setupUseCase.Execute(ctx, input)
	return err
}

