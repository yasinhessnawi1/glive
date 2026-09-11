package handlers

import (
	"github.com/glive/interface/api"
	"github.com/gofiber/fiber/v2"
)

// GetConfig returns the current configuration
func GetConfig(c *fiber.Ctx) error {
	// TODO: Implement actual config retrieval
	data := map[string]interface{}{
		"api_provider":   "deepseek",
		"default_mode":   "auto",
		"workspace_dir":  "~/glive-workspace",
		"agent_port":     8080,
		"enable_sandbox": false,
	}
	return api.SuccessResponse(c, fiber.StatusOK, data)
}

// UpdateConfig updates configuration
func UpdateConfig(c *fiber.Ctx) error {
	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return api.BadRequest(c, "INVALID_REQUEST_BODY", "Invalid request body", nil)
	}

	// TODO: Implement actual config update
	data := map[string]interface{}{
		"message": "Configuration updated",
		"updates": updates,
	}
	return api.SuccessResponse(c, fiber.StatusOK, data)
}
