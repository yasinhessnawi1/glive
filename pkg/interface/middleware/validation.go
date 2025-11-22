package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	domainerrors "github.com/glive/domain/errors"
	"github.com/glive/domain/values"
)

// RequestValidator interface for validating request bodies
type RequestValidator interface {
	Validate(body []byte) error
}

// ValidationMiddleware validates HTTP requests
type ValidationMiddleware struct {
	maxBodySize int64
	validators  map[string]RequestValidator
}

// NewValidationMiddleware creates a new validation middleware
func NewValidationMiddleware() *ValidationMiddleware {
	return &ValidationMiddleware{
		maxBodySize: 10 * 1024 * 1024, // 10MB
		validators:  make(map[string]RequestValidator),
	}
}

// RegisterValidator registers a validator for a specific path
func (m *ValidationMiddleware) RegisterValidator(path string, v RequestValidator) {
	m.validators[path] = v
}

// Middleware returns the HTTP middleware function
func (m *ValidationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate content type
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
			ct := r.Header.Get("Content-Type")
			if !strings.HasPrefix(ct, "application/json") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnsupportedMediaType)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Content-Type must be application/json",
					"code":  "INVALID_CONTENT_TYPE",
				})
				return
			}
		}

		// Limit body size
		r.Body = http.MaxBytesReader(w, r.Body, m.maxBodySize)

		// Read body if there's a validator for this path
		validator, exists := m.validators[r.URL.Path]
		if exists && r.Body != nil {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				if err.Error() == "http: request body too large" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusRequestEntityTooLarge)
					json.NewEncoder(w).Encode(map[string]string{
						"error": "Request body too large",
						"code":  "BODY_TOO_LARGE",
					})
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Failed to read request body",
					"code":  "READ_ERROR",
				})
				return
			}

			if err := validator.Validate(body); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				
				// Format error response
				var gliveErr *domainerrors.GliveError
				if errors.As(err, &gliveErr) {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"error":   gliveErr.Message,
						"code":    gliveErr.Code,
						"details": gliveErr.Context,
					})
				} else {
					json.NewEncoder(w).Encode(map[string]string{
						"error": err.Error(),
						"code":  "VALIDATION_ERROR",
					})
				}
				return
			}

			// Reset body for downstream handlers
			r.Body = io.NopCloser(bytes.NewBuffer(body))
		}

		next.ServeHTTP(w, r)
	})
}

// RunProjectValidator validates /run endpoint requests
type RunProjectValidator struct{}

// Validate validates a run project request
func (v *RunProjectValidator) Validate(body []byte) error {
	var req struct {
		URL  string `json:"url"`
		Mode string `json:"mode"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		return domainerrors.NewUserError("INVALID_JSON", "Invalid JSON in request body")
	}

	// Validate URL
	if req.URL == "" {
		return domainerrors.NewUserError("MISSING_URL", "URL is required")
	}

	if _, err := values.ParseRepoURL(req.URL); err != nil {
		return err
	}

	// Validate mode
	if req.Mode != "" {
		validModes := []string{"auto", "assisted", "manual"}
		valid := false
		for _, m := range validModes {
			if req.Mode == m {
				valid = true
				break
			}
		}
		if !valid {
			return domainerrors.NewUserError("INVALID_MODE",
				fmt.Sprintf("Mode must be one of: %v", validModes))
		}
	}

	return nil
}

// CreateProjectValidator validates project creation requests
type CreateProjectValidator struct{}

// Validate validates a create project request
func (v *CreateProjectValidator) Validate(body []byte) error {
	var req struct {
		GitHubURL string `json:"github_url"`
		Mode      string `json:"mode,omitempty"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		return domainerrors.NewUserError("INVALID_JSON", "Invalid JSON in request body")
	}

	// Validate URL
	if req.GitHubURL == "" {
		return domainerrors.NewUserError("MISSING_URL", "github_url is required")
	}

	if _, err := values.ParseRepoURL(req.GitHubURL); err != nil {
		return err
	}

	// Validate mode if provided
	if req.Mode != "" {
		validModes := []string{"auto", "assisted", "manual"}
		valid := false
		for _, m := range validModes {
			if req.Mode == m {
				valid = true
				break
			}
		}
		if !valid {
			return domainerrors.NewUserError("INVALID_MODE",
				fmt.Sprintf("Mode must be one of: %v", validModes))
		}
	}

	return nil
}

