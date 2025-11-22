package controller

import (
	"context"
	"io"
	"os"

	"github.com/glive/infrastructure/container"
	"github.com/glive/infrastructure/executor"
	"github.com/glive/interface/presenter"
	"github.com/glive/usecase/project"
)

// CLIController handles CLI input and delegates to use cases
type CLIController struct {
	setupUseCase *project.SetupProjectUseCase
	presenter    presenter.CLIPresenter
}

// NewCLIController creates a new CLI controller
func NewCLIController(cont *container.Container, output io.Writer) *CLIController {
	if output == nil {
		output = os.Stdout
	}

	projectRepo := cont.ProjectRepository()
	setupUC := project.NewSetupProjectUseCase(projectRepo, cont, output)

	return &CLIController{
		setupUseCase: setupUC,
		presenter:    presenter.NewCLIPresenter(output),
	}
}

// SetProgressCallback sets the progress callback for the setup use case
func (c *CLIController) SetProgressCallback(callback project.ProgressCallback) {
	c.setupUseCase.SetProgressCallback(callback)
}

// RunProject runs a project from a GitHub URL
func (c *CLIController) RunProject(ctx context.Context, githubURL string, mode executor.ExecutionMode, workspaceDir string, force bool) error {
	return c.RunProjectWithAIOptions(ctx, githubURL, mode, workspaceDir, force, "", 0.0, false, false)
}

// RunProjectWithAIOptions runs a project with AI-first options
func (c *CLIController) RunProjectWithAIOptions(ctx context.Context, githubURL string, mode executor.ExecutionMode, workspaceDir string, force bool, aiMode string, aiConfidence float64, noFallback, noAIRecovery bool) error {
	input := project.SetupProjectInput{
		GitHubURL:    githubURL,
		Mode:         mode,
		WorkspaceDir: workspaceDir,
		Force:        force,
		AIMode:       aiMode,
		AIConfidence: aiConfidence,
		NoFallback:   noFallback,
		NoAIRecovery: noAIRecovery,
	}

	output, err := c.setupUseCase.Execute(ctx, input)
	if err != nil {
		return c.presenter.PresentError(err)
	}

	return c.presenter.PresentSuccess(output)
}

