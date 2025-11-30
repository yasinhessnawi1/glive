package server

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/glive/agent/events"
	"github.com/glive/agent/handlers"
	"github.com/glive/core"
	"github.com/glive/core/state"
	"github.com/glive/infrastructure/tracing"
	"github.com/glive/interface/api"
	"github.com/glive/interface/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
)

// Server represents the GLive API server
type Server struct {
	app    *fiber.App
	config *core.Config
}

// New creates a new server instance
func New(config *core.Config) *Server {
	app := fiber.New(fiber.Config{
		AppName: "GLive Agent",
	})

	// Middleware
	app.Use(logger.New())

	// Tracing middleware - must be early to capture full request lifecycle
	// Always enabled for observability (lightweight)
	app.Use(tracing.FiberMiddleware())

	// CORS middleware - must be early to handle preflight requests
	corsOrigins := "http://localhost:3000,http://127.0.0.1:3000,https://g-live.vercel.app,https://glive-agent.fly.dev"
	if os.Getenv("GLIVE_ENV") == "production" {
		corsOrigins = "*" // Allow all origins in production (Vercel will have dynamic URLs)
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Requested-With",
		ExposeHeaders:    "API-Version",
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}))

	// Response context middleware - track request IDs and timing
	app.Use(func(c *fiber.Ctx) error {
		ctx := api.NewResponseContext()
		api.SetResponseContext(c, ctx)
		return c.Next()
	})

	// Security middleware - restrict to localhost only (skip OPTIONS and WebSocket upgrades)
	app.Use(func(c *fiber.Ctx) error {
		// Skip security check for OPTIONS requests (CORS preflight)
		if c.Method() == "OPTIONS" {
			return c.Next()
		}

		// Skip security check for WebSocket upgrade requests
		if c.Get("Upgrade") == "websocket" {
			return c.Next()
		}

		// Add security headers
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Content-Security-Policy", "default-src 'self'")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Check if request is from localhost
		// For CORS requests, the Origin header is checked by CORS middleware
		// Here we check the actual IP address
		// Skip check if running in production (assumes protected by other means or intended for public access)
		if os.Getenv("GLIVE_ENV") == "production" {
			return c.Next()
		}

		host, _, err := net.SplitHostPort(c.IP())
		if err != nil {
			host = c.IP()
		}

		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return c.Status(403).JSON(fiber.Map{
				"error": "Forbidden: Only local connections allowed",
			})
		}

		return c.Next()
	})

	// API versioning middleware - supports both URL and header-based versioning
	app.Use(api.VersionMiddleware())

	// Rate limiting - 100 requests per minute per IP
	rateLimiter := middleware.NewRateLimiter(100, 1*time.Minute)
	defer rateLimiter.Stop()

	// Initialize dependencies
	stateDir := filepath.Join(config.WorkspaceDir, ".glive")
	stateMgr, err := state.New(stateDir)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize state manager: %v", err))
	}

	orchestrator, err := core.NewOrchestrator(config, nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize orchestrator: %v", err))
	}

	eventBus := events.NewBus()

	// Create handler with dependencies
	h := handlers.New(stateMgr, orchestrator, eventBus)

	srv := &Server{
		app:    app,
		config: config,
	}

	srv.setupRoutes(rateLimiter, h)

	return srv
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes(rateLimiter *middleware.RateLimiter, h *handlers.Handler) {
	// API v1 routes with rate limiting
	api := s.app.Group("/api/v1", rateLimiter.Middleware())

	// Future: API v2 routes can be added here
	// apiV2 := s.app.Group("/api/v2", rateLimiter.Middleware())

	// Health check (no rate limiting)
	s.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"version": "0.1.0",
		})
	})

	// Metrics endpoint (no rate limiting) - for Prometheus scraping
	// Always available if metrics are initialized
	s.app.Get("/metrics", handlers.MetricsHandler)

	// Projects
	api.Get("/projects", h.ListProjects)
	api.Post("/projects", h.CreateProject)
	api.Get("/projects/:id", h.GetProject)
	api.Delete("/projects/:id", h.DeleteProject)
	api.Post("/projects/:id/start", h.StartProject)
	api.Post("/projects/:id/stop", h.StopProject)
	api.Post("/projects/:id/cleanup", h.CleanupProject)

	// Project downloads and tools
	api.Get("/projects/:id/download", h.DownloadProjectZip)
	api.Get("/projects/:id/vscode", h.GetVSCodeURL)
	api.Get("/projects/:id/report", h.GetExecutionReport)

	// Configuration
	api.Get("/config", handlers.GetConfig)
	api.Put("/config", handlers.UpdateConfig)

	// WebSocket for real-time updates
	api.Get("/ws/:project_id", websocket.New(h.HandleWebSocket))
}

// Start starts the server
func (s *Server) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	return s.app.Listen(addr)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	return s.app.Shutdown()
}
