package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/glive/core"
	"github.com/glive/interface/api"
	"github.com/gofiber/fiber/v2"
)

// CreateProjectRequest represents a request to create a project
type CreateProjectRequest struct {
	GitHubURL string `json:"github_url"`
	Mode      string `json:"mode"`
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

	// Start execution in background
	go func() {
		ctx := context.Background()
		// We ignore the returned project since we already have it, but we should handle errors
		_, err := h.Orchestrator.RunProject(ctx, projectID, req.GitHubURL, mode, func(update core.ProgressUpdate) {
			h.EventBus.Publish(update.ProjectID, update)
			statusMsg := api.NewProjectStatusMessage(update.ProjectID, update.Stage, update.Message)
			h.EventBus.Publish(update.ProjectID, statusMsg)
		})

		if err != nil {
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

	// Run in background to avoid blocking the request
	go func() {
		ctx := context.Background()
		err := h.Orchestrator.StartProject(ctx, projectID, func(update core.ProgressUpdate) {
			h.EventBus.Publish(update.ProjectID, update)
			statusMsg := api.NewProjectStatusMessage(update.ProjectID, update.Stage, update.Message)
			h.EventBus.Publish(update.ProjectID, statusMsg)
		})

		if err != nil {
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

	if err := h.Orchestrator.StopProject(projectID); err != nil {
		return api.InternalError(c, "STOP_FAILED", fmt.Sprintf("Failed to stop project: %v", err))
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
