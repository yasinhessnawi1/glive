package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

const (
	// APIVersionV1 represents API version 1
	APIVersionV1 = "v1"
	// APIVersionV2 represents API version 2 (future)
	APIVersionV2 = "v2"
	// DefaultAPIVersion is the default API version
	DefaultAPIVersion = APIVersionV1
)

// VersionContext holds the API version for the current request
type VersionContext struct {
	Version string
	Source  string // "url" or "header"
}

// GetAPIVersion extracts API version from Fiber context
// Checks both URL path (/api/v1/, /api/v2/) and Accept header
func GetAPIVersion(c *fiber.Ctx) *VersionContext {
	// First, check URL path
	path := c.Path()
	if strings.Contains(path, "/api/v2/") {
		return &VersionContext{
			Version: APIVersionV2,
			Source:  "url",
		}
	}
	if strings.Contains(path, "/api/v1/") {
		return &VersionContext{
			Version: APIVersionV1,
			Source:  "url",
		}
	}

	// Then check Accept header: application/vnd.glive.v1+json
	acceptHeader := c.Get("Accept")
	if acceptHeader != "" {
		if strings.Contains(acceptHeader, "vnd.glive.v2") {
			return &VersionContext{
				Version: APIVersionV2,
				Source:  "header",
			}
		}
		if strings.Contains(acceptHeader, "vnd.glive.v1") {
			return &VersionContext{
				Version: APIVersionV1,
				Source:  "header",
			}
		}
	}

	// Default to v1
	return &VersionContext{
		Version: DefaultAPIVersion,
		Source:  "default",
	}
}

// VersionMiddleware is a Fiber middleware that extracts and validates API version
func VersionMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		versionCtx := GetAPIVersion(c)
		c.Locals("api_version", versionCtx.Version)
		c.Locals("api_version_source", versionCtx.Source)

		// Set version header in response
		c.Set("API-Version", versionCtx.Version)

		return c.Next()
	}
}

// IsV1 checks if the request is for API v1
func IsV1(c *fiber.Ctx) bool {
	version := c.Locals("api_version")
	if version == nil {
		return true // default to v1
	}
	return version.(string) == APIVersionV1
}

// IsV2 checks if the request is for API v2
func IsV2(c *fiber.Ctx) bool {
	version := c.Locals("api_version")
	if version == nil {
		return false
	}
	return version.(string) == APIVersionV2
}
