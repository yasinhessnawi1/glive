package handlers

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glive/core"
	"github.com/glive/interface/api"
	"github.com/gofiber/fiber/v2"
)

// CreateProjectRequest represents a request to create a project
type CreateProjectRequest struct {
	GitHubURL      string `json:"github_url"`
	Mode           string `json:"mode"`
	ForceExecution bool   `json:"force_execution"`
}

// ListProjects returns all projects
func (h *Handler) ListProjects(c *fiber.Ctx) error {
	projects := h.StateManager.ListProjects()
	data := map[string]interface{}{
		"projects": projects,
	}
	return api.SuccessResponse(c, fiber.StatusOK, data)
}

// CreateProject creates a new project from a GitHub URL
func (h *Handler) CreateProject(c *fiber.Ctx) error {
	var req CreateProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return api.BadRequest(c, "INVALID_REQUEST_BODY", "Invalid request body", nil)
	}

	if req.GitHubURL == "" {
		return api.BadRequest(c, "MISSING_GITHUB_URL", "github_url is required", nil)
	}

	// Determine execution mode
	mode := core.ModeAuto
	if req.Mode == "manual" {
		mode = core.ModeManual
	} else if req.Mode == "assisted" {
		mode = core.ModeAssisted
	}

	// Generate Project ID
	projectID := fmt.Sprintf("project-%d", time.Now().UnixNano())

	// Create initial project state
	project := &core.Project{
		ID:        projectID,
		GitHubURL: req.GitHubURL,
		Status:    core.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to state manager immediately
	if err := h.StateManager.SaveProject(project); err != nil {
		return api.InternalError(c, "STATE_SAVE_FAILED", "Failed to save project state")
	}

	// Create cancellable context for the execution
	ctx, cancel := context.WithCancel(context.Background())

	// Register the cancel function so StopProject can cancel this execution
	h.Orchestrator.RegisterRunningProject(projectID, cancel)

	// Clear any old buffered events before starting
	h.EventBus.ClearBuffer(projectID)

	// Start execution in background immediately
	// Events are buffered so client can receive them on reconnect
	go func() {
		// Ensure we clean up when done
		defer h.Orchestrator.UnregisterRunningProject(projectID)

		// We ignore the returned project since we already have it, but we should handle errors
		_, err := h.Orchestrator.RunProject(ctx, projectID, req.GitHubURL, mode, req.ForceExecution, func(update core.ProgressUpdate) {
			// Only buffer meaningful stage changes for report generation (not every log line)
			// Buffer if:
			// 1. It's a major stage (parsing, cloning, scanning, analyzing, installing, executing)
			// 2. It's a success notification (Success = true)
			// 3. Stage is empty (command_started, command_complete events)
			meaningfulStages := map[string]bool{
				"parsing":    true,
				"cloning":    true,
				"scanning":   true,
				"analyzing":  true,
				"installing": true,
				"executing":  true,
			}

			shouldBuffer := meaningfulStages[update.Stage] || update.Success || update.Stage == ""

			if shouldBuffer {
				// Publish with buffering for meaningful events (used in report generation)
				h.EventBus.Publish(update.ProjectID, update)
			} else {
				// Publish without buffering for log lines (WebSocket subscribers still see them)
				h.EventBus.PublishWithoutBuffer(update.ProjectID, update)
			}
		})

		if err != nil {
			// Check if it was cancelled (user stopped)
			if ctx.Err() == context.Canceled {
				project.Status = core.StatusStopped
				h.StateManager.SaveProject(project)
				statusMsg := api.NewProjectStatusMessage(projectID, "stopped", "Project execution stopped by user")
				h.EventBus.Publish(projectID, statusMsg)
				return
			}

			// Update status to failed
			project.Status = core.StatusFailed
			h.StateManager.SaveProject(project)

			// Broadcast error
			errorMsg := api.NewErrorMessage("EXECUTION_FAILED", err.Error(), nil)
			h.EventBus.Publish(projectID, errorMsg)
		}
	}()

	// Return immediately with the project details
	return api.SuccessResponse(c, fiber.StatusCreated, project)
}

// GetProject returns a specific project
func (h *Handler) GetProject(c *fiber.Ctx) error {
	projectID := c.Params("id")

	project, err := h.StateManager.LoadProject(projectID)
	if err != nil {
		return api.NotFound(c, "PROJECT_NOT_FOUND", "Project not found")
	}

	return api.SuccessResponse(c, fiber.StatusOK, project)
}

// DeleteProject deletes a project
func (h *Handler) DeleteProject(c *fiber.Ctx) error {
	projectID := c.Params("id")

	if err := h.StateManager.DeleteProject(projectID); err != nil {
		return api.InternalError(c, "DELETE_FAILED", "Failed to delete project")
	}

	data := map[string]interface{}{
		"message": "Project deleted",
		"id":      projectID,
	}
	return api.SuccessResponse(c, fiber.StatusOK, data)
}

// StartProject starts a project
func (h *Handler) StartProject(c *fiber.Ctx) error {
	projectID := c.Params("id")

	fmt.Printf("🚀 StartProject called for project: %s\n", projectID)

	// Run in background to avoid blocking the request
	go func() {
		ctx := context.Background()

		err := h.Orchestrator.StartProject(ctx, projectID, func(update core.ProgressUpdate) {
			// Only publish the ProgressUpdate - the websocket handler will convert it appropriately
			h.EventBus.Publish(update.ProjectID, update)
		})

		if err != nil {
			fmt.Printf("❌ StartProject error for %s: %v\n", projectID, err)
			errorMsg := api.NewErrorMessage("START_FAILED", err.Error(), nil)
			h.EventBus.Publish(projectID, errorMsg)
		}
	}()

	data := map[string]interface{}{
		"message": "Project start initiated",
		"id":      projectID,
	}
	return api.SuccessResponse(c, fiber.StatusOK, data)
}

// StopProject stops a running project
func (h *Handler) StopProject(c *fiber.Ctx) error {
	projectID := c.Params("id")

	// Try to stop through orchestrator (this will cancel the context)
	err := h.Orchestrator.StopProject(projectID)
	if err != nil {
		// If the error is "project is not running", we still want to update the status
		// This can happen if the project finished between status checks
		// Load project and update status anyway
		project, loadErr := h.StateManager.LoadProject(projectID)
		if loadErr != nil {
			return api.NotFound(c, "PROJECT_NOT_FOUND", "Project not found")
		}

		// If project is in an active state, force update to stopped
		activeStatuses := map[core.ProjectStatus]bool{
			core.StatusRunning:    true,
			core.StatusCloning:    true,
			core.StatusAnalyzing:  true,
			core.StatusInstalling: true,
			core.StatusPending:    true,
		}

		if activeStatuses[project.Status] {
			project.Status = core.StatusStopped
			h.StateManager.SaveProject(project)

			// Broadcast stopped status
			statusMsg := api.NewProjectStatusMessage(projectID, "stopped", "Project stopped by user")
			h.EventBus.Publish(projectID, statusMsg)
		}
	}

	data := map[string]interface{}{
		"message": "Project stopped",
		"id":      projectID,
	}
	return api.SuccessResponse(c, fiber.StatusOK, data)
}

// CleanupProject cleans up a project
func (h *Handler) CleanupProject(c *fiber.Ctx) error {
	projectID := c.Params("id")

	if err := h.Orchestrator.CleanupProject(projectID); err != nil {
		return api.InternalError(c, "CLEANUP_FAILED", fmt.Sprintf("Failed to cleanup project: %v", err))
	}

	data := map[string]interface{}{
		"message": "Project cleanup initiated",
		"id":      projectID,
	}
	return api.SuccessResponse(c, fiber.StatusOK, data)
}

// DownloadProjectZip creates and streams a ZIP file of the project
func (h *Handler) DownloadProjectZip(c *fiber.Ctx) error {
	projectID := c.Params("id")

	project, err := h.StateManager.LoadProject(projectID)
	if err != nil {
		return api.NotFound(c, "PROJECT_NOT_FOUND", "Project not found")
	}

	if project.LocalPath == "" {
		return api.BadRequest(c, "NO_LOCAL_PATH", "Project has no local path - it may not have been cloned yet", nil)
	}

	// Check if directory exists
	if _, err := os.Stat(project.LocalPath); os.IsNotExist(err) {
		return api.NotFound(c, "PROJECT_DIR_NOT_FOUND", "Project directory not found")
	}

	// Set response headers for ZIP download
	fileName := fmt.Sprintf("%s.zip", project.Name)
	c.Set("Content-Type", "application/zip")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))

	// Create a pipe to stream the ZIP
	pr, pw := io.Pipe()

	// Create ZIP in a goroutine to allow streaming
	go func() {
		defer pw.Close()
		zipWriter := zip.NewWriter(pw)
		defer zipWriter.Close()

		// Directories to skip (common large/unneeded directories)
		skipDirs := map[string]bool{
			"node_modules": true,
			".git":         true,
			"__pycache__":  true,
			".venv":        true,
			"venv":         true,
			"vendor":       true,
			"target":       true, // Rust
			"dist":         true,
			"build":        true,
			".next":        true,
			".nuxt":        true,
		}

		filepath.Walk(project.LocalPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip files we can't access
			}

			// Get relative path
			relPath, err := filepath.Rel(project.LocalPath, path)
			if err != nil {
				return nil
			}

			// Skip root
			if relPath == "." {
				return nil
			}

			// Check if this is a directory we should skip
			if info.IsDir() {
				if skipDirs[info.Name()] {
					return filepath.SkipDir
				}
				return nil
			}

			// Skip very large files (>50MB)
			if info.Size() > 50*1024*1024 {
				return nil
			}

			// Create ZIP entry
			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return nil
			}
			header.Name = filepath.ToSlash(relPath)
			header.Method = zip.Deflate

			writer, err := zipWriter.CreateHeader(header)
			if err != nil {
				return nil
			}

			// Copy file content
			file, err := os.Open(path)
			if err != nil {
				return nil
			}
			defer file.Close()

			io.Copy(writer, file)
			return nil
		})
	}()

	return c.SendStream(pr)
}

// GetVSCodeURL returns URLs for opening the project in VS Code
func (h *Handler) GetVSCodeURL(c *fiber.Ctx) error {
	projectID := c.Params("id")

	project, err := h.StateManager.LoadProject(projectID)
	if err != nil {
		return api.NotFound(c, "PROJECT_NOT_FOUND", "Project not found")
	}

	if project.LocalPath == "" {
		return api.BadRequest(c, "NO_LOCAL_PATH", "Project has no local path", nil)
	}

	// Generate different VS Code URLs
	// 1. Local VS Code using vscode:// protocol
	localVSCodeURL := fmt.Sprintf("vscode://file/%s", filepath.ToSlash(project.LocalPath))

	// 2. VS Code Web (vscode.dev) - for GitHub repos
	var webVSCodeURL string
	if project.GitHubURL != "" {
		// Convert github.com URL to vscode.dev URL
		// https://github.com/owner/repo -> https://vscode.dev/github/owner/repo
		githubURL := project.GitHubURL
		githubURL = strings.TrimSuffix(githubURL, ".git")
		githubURL = strings.Replace(githubURL, "https://github.com/", "", 1)
		githubURL = strings.Replace(githubURL, "http://github.com/", "", 1)
		webVSCodeURL = fmt.Sprintf("https://vscode.dev/github/%s", githubURL)
	}

	// 3. GitHub.dev (alternative web editor)
	var githubDevURL string
	if project.GitHubURL != "" {
		githubDevURL = strings.Replace(project.GitHubURL, "github.com", "github.dev", 1)
		githubDevURL = strings.TrimSuffix(githubDevURL, ".git")
	}

	data := map[string]interface{}{
		"local_vscode_url": localVSCodeURL,
		"web_vscode_url":   webVSCodeURL,
		"github_dev_url":   githubDevURL,
		"project_path":     project.LocalPath,
	}

	return api.SuccessResponse(c, fiber.StatusOK, data)
}

// ExecutionStep represents a step in the execution report
type ExecutionStep struct {
	Step        int       `json:"step"`
	Stage       string    `json:"stage"`
	Command     string    `json:"command,omitempty"`
	Description string    `json:"description"`
	Output      string    `json:"output,omitempty"`
	Duration    string    `json:"duration,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Success     bool      `json:"success"`
}

// ExecutionReport represents a full execution report
type ExecutionReport struct {
	ProjectID    string          `json:"project_id"`
	ProjectName  string          `json:"project_name"`
	GitHubURL    string          `json:"github_url"`
	LocalPath    string          `json:"local_path"`
	ProjectType  string          `json:"project_type"`
	Status       string          `json:"status"`
	CreatedAt    time.Time       `json:"created_at"`
	CompletedAt  time.Time       `json:"completed_at,omitempty"`
	TotalDuration string         `json:"total_duration,omitempty"`
	Steps        []ExecutionStep `json:"steps"`
	Summary      string          `json:"summary"`
	NextSteps    []string        `json:"next_steps"`
}

// GetExecutionReport generates a detailed execution report for a project
func (h *Handler) GetExecutionReport(c *fiber.Ctx) error {
	projectID := c.Params("id")
	format := c.Query("format", "json") // json or markdown

	project, err := h.StateManager.LoadProject(projectID)
	if err != nil {
		return api.NotFound(c, "PROJECT_NOT_FOUND", "Project not found")
	}

	// Build execution steps from buffered events
	bufferedEvents := h.EventBus.GetBufferedEvents(projectID)

	var steps []ExecutionStep
	stepNum := 1

	// Only include meaningful stages (not "log", "running", etc.)
	meaningfulStages := map[string]bool{
		"parsing":           true,
		"cloning":           true,
		"scanning":          true,
		"analyzing":         true,
		"ai_analysis":       true,
		"installing":        true,
		"executing":         true,
		"command_started":   true,
		"command_complete":  true,
		"execution_completed": true,
	}

	for _, event := range bufferedEvents {
		if update, ok := event.(core.ProgressUpdate); ok {
			// Filter out non-meaningful stages
			if !meaningfulStages[update.Stage] && update.Stage != "" {
				continue
			}

			step := ExecutionStep{
				Step:        stepNum,
				Stage:       update.Stage,
				Description: update.Message,
				Timestamp:   update.Timestamp,
				Success:     update.Success,
			}

			if update.Command != "" {
				step.Command = update.Command
			}
			if update.Duration > 0 {
				step.Duration = fmt.Sprintf("%v", update.Duration)
			}

			steps = append(steps, step)
			stepNum++
		}
	}

	// Calculate total duration
	var totalDuration string
	if !project.CreatedAt.IsZero() && project.Status == core.StatusReady {
		totalDuration = time.Since(project.CreatedAt).Round(time.Second).String()
	}

	// Generate summary based on status
	var summary string
	switch project.Status {
	case core.StatusReady:
		summary = fmt.Sprintf("Project '%s' was successfully cloned, analyzed, and set up. All dependencies have been installed and the project is ready to run.", project.Name)
	case core.StatusRunning:
		summary = fmt.Sprintf("Project '%s' is currently running.", project.Name)
	case core.StatusFailed:
		summary = fmt.Sprintf("Project '%s' setup failed. Please review the steps above to identify the issue.", project.Name)
	case core.StatusStopped:
		summary = fmt.Sprintf("Project '%s' was stopped by the user.", project.Name)
	default:
		summary = fmt.Sprintf("Project '%s' is in %s state.", project.Name, project.Status)
	}

	// Generate next steps
	nextSteps := []string{}
	switch project.Status {
	case core.StatusReady:
		nextSteps = append(nextSteps, fmt.Sprintf("Navigate to project directory: cd %s", project.LocalPath))
		if project.Type == "nodejs" {
			nextSteps = append(nextSteps, "Run the project: npm start or npm run dev")
		} else if project.Type == "python" {
			nextSteps = append(nextSteps, "Run the project: python main.py or python app.py")
		} else if project.Type == "go" {
			nextSteps = append(nextSteps, "Run the project: go run . or go run main.go")
		}
		nextSteps = append(nextSteps, "Open in VS Code: code .")
	case core.StatusFailed:
		nextSteps = append(nextSteps, "Review the error messages in the execution steps above")
		nextSteps = append(nextSteps, "Check if all required dependencies are available")
		nextSteps = append(nextSteps, "Try running the setup manually to get more detailed error output")
	}

	report := ExecutionReport{
		ProjectID:     project.ID,
		ProjectName:   project.Name,
		GitHubURL:     project.GitHubURL,
		LocalPath:     project.LocalPath,
		ProjectType:   string(project.Type),
		Status:        string(project.Status),
		CreatedAt:     project.CreatedAt,
		TotalDuration: totalDuration,
		Steps:         steps,
		Summary:       summary,
		NextSteps:     nextSteps,
	}

	// Return based on format
	if format == "markdown" {
		markdown := generateMarkdownReport(report)
		c.Set("Content-Type", "text/markdown")
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fmt.Sprintf("%s-report.md", project.Name)))
		return c.SendString(markdown)
	}

	return api.SuccessResponse(c, fiber.StatusOK, report)
}

// generateMarkdownReport creates a markdown formatted report
func generateMarkdownReport(report ExecutionReport) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# GLive Execution Report: %s\n\n", report.ProjectName))
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format(time.RFC1123)))

	sb.WriteString("## Project Information\n\n")
	sb.WriteString(fmt.Sprintf("- **Project ID:** %s\n", report.ProjectID))
	sb.WriteString(fmt.Sprintf("- **GitHub URL:** %s\n", report.GitHubURL))
	sb.WriteString(fmt.Sprintf("- **Local Path:** `%s`\n", report.LocalPath))
	sb.WriteString(fmt.Sprintf("- **Project Type:** %s\n", report.ProjectType))
	sb.WriteString(fmt.Sprintf("- **Status:** %s\n", report.Status))
	if report.TotalDuration != "" {
		sb.WriteString(fmt.Sprintf("- **Total Duration:** %s\n", report.TotalDuration))
	}
	sb.WriteString("\n")

	sb.WriteString("## Summary\n\n")
	sb.WriteString(report.Summary + "\n\n")

	if len(report.Steps) > 0 {
		sb.WriteString("## Execution Steps\n\n")
		for _, step := range report.Steps {
			sb.WriteString(fmt.Sprintf("### Step %d: %s\n\n", step.Step, step.Stage))
			sb.WriteString(fmt.Sprintf("- **Description:** %s\n", step.Description))
			if step.Command != "" {
				sb.WriteString(fmt.Sprintf("- **Command:** `%s`\n", step.Command))
			}
			if step.Duration != "" {
				sb.WriteString(fmt.Sprintf("- **Duration:** %s\n", step.Duration))
			}
			sb.WriteString(fmt.Sprintf("- **Status:** %s\n", map[bool]string{true: "✓ Success", false: "✗ Failed"}[step.Success]))
			if step.Output != "" {
				sb.WriteString(fmt.Sprintf("\n```\n%s\n```\n", step.Output))
			}
			sb.WriteString("\n")
		}
	}

	if len(report.NextSteps) > 0 {
		sb.WriteString("## Next Steps\n\n")
		sb.WriteString("To run this project on your machine:\n\n")
		for i, step := range report.NextSteps {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## How to Reproduce\n\n")
	sb.WriteString("To achieve the same setup manually, follow these exact steps that GLive executed:\n\n")
	sb.WriteString(fmt.Sprintf("1. Clone the repository:\n   ```bash\n   git clone %s\n   cd %s\n   ```\n\n", report.GitHubURL, report.ProjectName))

	// Extract actual commands from execution steps
	commandSteps := []string{}
	for _, step := range report.Steps {
		if step.Command != "" && step.Success {
			// Skip certain system commands that aren't user-relevant
			if step.Command != "git clone" && step.Command != "cd" {
				commandSteps = append(commandSteps, fmt.Sprintf("   ```bash\n   %s\n   ```\n   *%s*", step.Command, step.Description))
			}
		}
	}

	if len(commandSteps) > 0 {
		sb.WriteString("2. Execute the following commands (as run by GLive):\n\n")
		for _, cmd := range commandSteps {
			sb.WriteString(cmd + "\n\n")
		}
	} else {
		// Fallback to project type-based instructions if no commands were captured
		switch report.ProjectType {
		case "nodejs":
			sb.WriteString("2. Install dependencies:\n   ```bash\n   npm install\n   # or: yarn install / pnpm install\n   ```\n\n")
			sb.WriteString("3. Run the project:\n   ```bash\n   npm start\n   # or: npm run dev\n   ```\n\n")
		case "python":
			sb.WriteString("2. Create virtual environment (recommended):\n   ```bash\n   python -m venv venv\n   source venv/bin/activate  # Linux/Mac\n   # or: venv\\Scripts\\activate  # Windows\n   ```\n\n")
			sb.WriteString("3. Install dependencies:\n   ```bash\n   pip install -r requirements.txt\n   ```\n\n")
			sb.WriteString("4. Run the project:\n   ```bash\n   python main.py\n   # or: python app.py\n   ```\n\n")
		case "go":
			sb.WriteString("2. Download dependencies:\n   ```bash\n   go mod download\n   ```\n\n")
			sb.WriteString("3. Run the project:\n   ```bash\n   go run .\n   # or: go run main.go\n   ```\n\n")
		default:
			sb.WriteString("2. Install dependencies and run the project:\n   ```bash\n   # Check the Execution Steps section above for the exact commands GLive used\n   ```\n\n")
		}
	}

	// Add AI assumptions and analysis if available
	sb.WriteString("### AI Analysis & Assumptions\n\n")
	sb.WriteString(fmt.Sprintf("- **Detected Project Type:** %s\n", report.ProjectType))
	if report.Summary != "" {
		sb.WriteString(fmt.Sprintf("- **Analysis:** %s\n", report.Summary))
	}
	sb.WriteString("\n*Note: GLive analyzed the project structure and dependencies to determine the optimal setup approach.*\n\n")

	sb.WriteString("---\n")
	sb.WriteString("*Report generated by [GLive](https://github.com/yourusername/glive)*\n")

	return sb.String()
}
