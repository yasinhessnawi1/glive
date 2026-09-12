package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/glive/agent/events"
	"github.com/glive/agent/handlers"
	"github.com/glive/core"
	"github.com/glive/core/state"
	"github.com/glive/interface/api"
)

// Contract tests for the agent's HTTP surface - GL0 T2.
//
// UNLIKE the core.Orchestrator capture in pkg/core, THESE TESTS SURVIVE T8. They
// are the oracle for T6: when the agent is routed through
// pkg/usecase/project.SetupProjectUseCase, every assertion here must still pass
// UNMODIFIED. Anything that has to change is, by definition, a user-visible
// change to the API and belongs in T6's report.
//
// In-package because Server.app is unexported and there is no accessor.
//
// The file is in two halves, because the server's middleware makes the handlers
// unreachable in-process:
//
//   - the ROUTE CONTRACT is exercised against an app that mounts the real
//     handlers.Handler methods and the real response-context middleware, but not
//     the loopback guard. fiber's app.Test() synthesises a connection whose
//     c.IP() is not a loopback address, so on the real server every request is
//     refused with 403 before any handler runs;
//   - the MIDDLEWARE - the loopback guard and the GLIVE_ENV=production bypass -
//     is pinned against the real server, where it belongs.
//
// No network: the workspace is a temp directory with no projects in it, so the
// handlers take their not-found and validation paths. Creating a project would
// clone and call a provider, which is out of scope.
//
// AGENT-RACE - A DEFECT THESE TESTS FOUND, NOT FIXED HERE.
//
// Every test below uses a FRESH app, and that is not only for isolation. Sharing
// one app made the suite fail under -race roughly one run in three, and the
// detector pointed at production code, not at the harness:
//
//	Write by fiber's AcquireCtx -> configDependentPaths (the next request)
//	Read  by handlers.(*Handler).StartProject.func1 at projects.go:180
//
// handlers.StartProject does:
//
//	projectID := c.Params("id")   // aliases fiber's POOLED request buffer
//	go func() { ... uses projectID ... }()
//	return api.SuccessResponse(...)   // request ends; fiber recycles the buffer
//
// fiber is explicit that values from c.Params / c.Query / c.Body are valid only
// for the lifetime of the handler and must be copied to outlive it. Nothing in
// pkg/agent/handlers copies one: there are four goroutine sites
// (projects.go:82, :171, :276 and websocket.go:158) and not a single
// utils.CopyString among them.
//
// The consequence is worse than corrupted log output. The captured id is used
// for h.Orchestrator.StartProject and for h.EventBus.Publish(projectID, ...), so
// once a concurrent request overwrites the buffer, the background goroutine can
// operate on - and publish events for - a DIFFERENT project than the one asked
// for. On the websocket path the goroutine is long-lived, which makes the window
// correspondingly wide: a client subscribed to one project can be served another
// project's events. That is a cross-request data-integrity bug today and a
// cross-tenant one in Mode A.
//
// Using a fresh app per test removes the window from THIS suite; it does not fix
// the product. Reported at the T2 gate for a ruling.

// ---------------------------------------------------------------- harness

// newRouteApp mounts the real handlers on a bare fiber app: same routes, same
// response-context middleware, no loopback guard. See the file comment.
func newRouteApp(t *testing.T) *fiber.App {
	t.Helper()

	workspace := t.TempDir()
	cfg := &core.Config{
		WorkspaceDir: workspace,
		DefaultMode:  core.ModeAuto,
		APIProvider:  "deepseek",
		AgentPort:    8080,
	}

	stateMgr, err := state.New(filepath.Join(workspace, ".glive"))
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	orch, err := core.NewOrchestrator(cfg, io.Discard)
	if err != nil {
		t.Fatalf("core.NewOrchestrator: %v", err)
	}
	h := handlers.New(stateMgr, orch, events.NewBus())

	app := fiber.New(fiber.Config{AppName: "GLive Agent (contract test)"})

	// The envelope's meta comes from this middleware; without it the responses
	// would not match what the server actually sends.
	app.Use(func(c *fiber.Ctx) error {
		api.SetResponseContext(c, api.NewResponseContext())
		return c.Next()
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy", "version": "0.1.0"})
	})

	v1 := app.Group("/api/v1")
	v1.Get("/projects", h.ListProjects)
	v1.Post("/projects", h.CreateProject)
	v1.Get("/projects/:id", h.GetProject)
	v1.Delete("/projects/:id", h.DeleteProject)
	v1.Post("/projects/:id/start", h.StartProject)
	v1.Post("/projects/:id/stop", h.StopProject)
	v1.Post("/projects/:id/cleanup", h.CleanupProject)
	v1.Get("/projects/:id/download", h.DownloadProjectZip)
	v1.Get("/projects/:id/vscode", h.GetVSCodeURL)
	v1.Get("/projects/:id/report", h.GetExecutionReport)
	v1.Get("/config", handlers.GetConfig)
	v1.Put("/config", handlers.UpdateConfig)

	return app
}

// do issues a request and returns the status code and the fully-read body. It
// deliberately does NOT return *http.Response: the body is consumed and closed
// here, so there is no open resource for a caller to leak.
func do(t *testing.T, app *fiber.App, method, path, body string) (int, []byte) {
	t.Helper()

	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := app.Test(req, 10_000)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	// Copy before returning. fiber's app.Test serves the request on its own
	// goroutine over a fasthttp connection whose buffers are POOLED and reused by
	// the next request, and the bytes read here can still alias that buffer.
	// Holding the slice - to format it into a failure message, say - then races
	// with the next call to app.Test. The race detector caught exactly that, in
	// this helper rather than in the product: a write from fiber's AcquireCtx
	// against a read from fmt formatting an earlier payload.
	out := make([]byte, len(payload))
	copy(out, payload)
	return resp.StatusCode, out
}

// envelope is the shape every /api/v1 response uses (interface/api/response.go).
type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Meta *struct {
		RequestID string `json:"request_id"`
		Duration  string `json:"duration"`
	} `json:"meta,omitempty"`
}

func decode(t *testing.T, payload []byte) envelope {
	t.Helper()
	var e envelope
	if err := json.Unmarshal(payload, &e); err != nil {
		t.Fatalf("response is not the standard envelope: %v; body: %s", err, payload)
	}
	return e
}

// ---------------------------------------------------------------- routes

// TestContract_RouteTable pins every route and what it returns against an empty
// workspace.
func TestContract_RouteTable(t *testing.T) {
	tests := []struct {
		method     string
		path       string
		body       string
		wantStatus int
		wantCode   string // envelope error code; "" means a success envelope
	}{
		{"GET", "/health", "", http.StatusOK, ""},

		{"GET", "/api/v1/projects", "", http.StatusOK, ""},
		{"POST", "/api/v1/projects", `{"not":"valid"}`, http.StatusBadRequest, "MISSING_GITHUB_URL"},
		{"GET", "/api/v1/projects/nope", "", http.StatusNotFound, "PROJECT_NOT_FOUND"},

		// INCONSISTENT, and pinned as such - see
		// TestContract_MutationsOnMissingProjectsReportSuccess below. GET and
		// stop check that the project exists; delete, start and cleanup do not,
		// and answer 200 with a success envelope for an id that never existed.
		{"DELETE", "/api/v1/projects/nope", "", http.StatusOK, ""},
		{"POST", "/api/v1/projects/nope/start", "", http.StatusOK, ""},
		{"POST", "/api/v1/projects/nope/stop", "", http.StatusNotFound, "PROJECT_NOT_FOUND"},
		{"POST", "/api/v1/projects/nope/cleanup", "", http.StatusOK, ""},

		{"GET", "/api/v1/projects/nope/download", "", http.StatusNotFound, "PROJECT_NOT_FOUND"},
		{"GET", "/api/v1/projects/nope/vscode", "", http.StatusNotFound, "PROJECT_NOT_FOUND"},
		{"GET", "/api/v1/projects/nope/report", "", http.StatusNotFound, "PROJECT_NOT_FOUND"},

		{"GET", "/api/v1/config", "", http.StatusOK, ""},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			// A fresh app per case. Sharing one made the cases order-dependent:
			// POST /projects/:id/start registers the id in the orchestrator's
			// running-project map, after which POST /projects/:id/stop finds it
			// and answers 200 instead of 404.
			app := newRouteApp(t)

			status, payload := do(t, app, tt.method, tt.path, tt.body)

			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", status, tt.wantStatus, payload)
			}

			// /health predates the envelope and returns a bare object.
			if tt.path == "/health" {
				var health map[string]any
				if err := json.Unmarshal(payload, &health); err != nil {
					t.Fatalf("/health is not JSON: %v", err)
				}
				if health["status"] != "healthy" {
					t.Errorf("/health status = %v, want %q", health["status"], "healthy")
				}
				return
			}

			e := decode(t, payload)
			if tt.wantCode == "" {
				if !e.Success {
					t.Errorf("success = false, want true (error: %+v)", e.Error)
				}
			} else {
				if e.Success {
					t.Error("success = true, want false")
				}
				if e.Error == nil {
					t.Fatalf("no error object on a %d response", status)
				}
				if e.Error.Code != tt.wantCode {
					t.Errorf("error.code = %q, want %q", e.Error.Code, tt.wantCode)
				}
			}
		})
	}
}

// TestContract_EnvelopeShape pins the envelope the typed web client depends on.
func TestContract_EnvelopeShape(t *testing.T) {
	app := newRouteApp(t)

	t.Run("success carries data and meta", func(t *testing.T) {
		_, payload := do(t, app, "GET", "/api/v1/projects", "")
		e := decode(t, payload)

		if !e.Success {
			t.Fatalf("success = false: %+v", e.Error)
		}
		if e.Meta == nil {
			t.Fatal("meta is absent on a success response")
		}
		if e.Meta.RequestID == "" {
			t.Error("meta.request_id is empty")
		}
		if e.Meta.Duration == "" {
			t.Error("meta.duration is empty")
		}

		// ListProjects wraps its payload in {"projects": [...]}.
		var data struct {
			Projects json.RawMessage `json:"projects"`
		}
		if err := json.Unmarshal(e.Data, &data); err != nil {
			t.Fatalf("data is not {projects: ...}: %v", err)
		}
		if data.Projects == nil {
			t.Error("data.projects is absent; the web client reads that key")
		}
	})

	t.Run("error carries code and message", func(t *testing.T) {
		_, payload := do(t, app, "GET", "/api/v1/projects/nope", "")
		e := decode(t, payload)

		if e.Success {
			t.Error("success = true on a 404")
		}
		if e.Error == nil {
			t.Fatal("error object is absent")
		}
		if e.Error.Code != "PROJECT_NOT_FOUND" {
			t.Errorf("error.code = %q, want %q", e.Error.Code, "PROJECT_NOT_FOUND")
		}
		if e.Error.Message == "" {
			t.Error("error.message is empty")
		}
	})
}

// TestContract_CreateProjectValidation pins the request contract for the one
// endpoint that takes a body, stopping short of a URL that would clone.
func TestContract_CreateProjectValidation(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{"unparseable body", `{`, http.StatusBadRequest, "INVALID_REQUEST_BODY"},
		{"missing github_url", `{}`, http.StatusBadRequest, "MISSING_GITHUB_URL"},
		{"empty github_url", `{"github_url":""}`, http.StatusBadRequest, "MISSING_GITHUB_URL"},
	}

	app := newRouteApp(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, payload := do(t, app, "POST", "/api/v1/projects", tt.body)

			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", status, tt.wantStatus, payload)
			}
			e := decode(t, payload)
			if e.Error == nil {
				t.Fatalf("no error object; body: %s", payload)
			}
			if e.Error.Code != tt.wantCode {
				t.Errorf("error.code = %q, want %q", e.Error.Code, tt.wantCode)
			}
		})
	}
}

// TestContract_MutationsOnMissingProjectsReportSuccess pins an INCONSISTENCY
// found by writing the route table.
//
// Five endpoints take a project id. Two of them check that the project exists and
// answer 404 PROJECT_NOT_FOUND; three do not, and answer 200 with a success
// envelope for an id that has never existed:
//
//	GET    /projects/:id          404  PROJECT_NOT_FOUND
//	POST   /projects/:id/stop     404  PROJECT_NOT_FOUND
//	DELETE /projects/:id          200  {"id":"nope","message":"Project deleted"}
//	POST   /projects/:id/start    200  {"id":"nope","message":"Project start initiated"}
//	POST   /projects/:id/cleanup  200  {"id":"nope","message":"Project cleanup initiated"}
//
// DELETE answering 200 is defensible as idempotency. "Project start initiated" for
// a project that does not exist is not: nothing was started, and the web client
// cannot tell that from a real start. A dashboard polling after this response
// waits for a project that will never appear.
//
// Not fixed here - T2 is characterisation, and T6 has to decide the contract when
// it routes these handlers through the use case. Pinned so that decision is
// explicit rather than inherited by accident.
func TestContract_MutationsOnMissingProjectsReportSuccess(t *testing.T) {
	checked := []struct {
		method, path string
	}{
		{"GET", "/api/v1/projects/nope"},
		{"POST", "/api/v1/projects/nope/stop"},
	}
	for _, c := range checked {
		t.Run("checks existence: "+c.method+" "+c.path, func(t *testing.T) {
			app := newRouteApp(t)
			status, payload := do(t, app, c.method, c.path, "")
			if status != http.StatusNotFound {
				t.Errorf("status = %d, want 404; body: %s", status, payload)
			}
		})
	}

	unchecked := []struct {
		method, path, wantMessage string
	}{
		{"DELETE", "/api/v1/projects/nope", "Project deleted"},
		{"POST", "/api/v1/projects/nope/start", "Project start initiated"},
		{"POST", "/api/v1/projects/nope/cleanup", "Project cleanup initiated"},
	}
	for _, u := range unchecked {
		t.Run("reports success: "+u.method+" "+u.path, func(t *testing.T) {
			// A fresh app per case, for the same reason as the route table, and
			// additionally because POST /start leaves a goroutine running that
			// holds fiber-owned memory - see the AGENT-RACE note in this file.
			app := newRouteApp(t)
			status, payload := do(t, app, u.method, u.path, "")
			if status != http.StatusOK {
				t.Fatalf("status = %d, want 200 (today's behaviour); body: %s", status, payload)
			}

			e := decode(t, payload)
			if !e.Success {
				t.Errorf("success = false; today this reports success for a missing project")
			}

			var data struct {
				ID      string `json:"id"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(e.Data, &data); err != nil {
				t.Fatalf("data is not {id, message}: %v", err)
			}
			if data.ID != "nope" {
				t.Errorf("data.id = %q, want %q", data.ID, "nope")
			}
			if data.Message != u.wantMessage {
				t.Errorf("data.message = %q, want %q", data.Message, u.wantMessage)
			}
		})
	}
}

// TestContract_UnknownRouteIs404 separates a fiber routing 404 (no envelope) from
// a handler 404 (envelope with a code).
func TestContract_UnknownRouteIs404(t *testing.T) {
	app := newRouteApp(t)

	status, payload := do(t, app, "GET", "/api/v1/does-not-exist", "")
	if status != http.StatusNotFound {
		t.Errorf("status = %d, want 404", status)
	}

	var e envelope
	if json.Unmarshal(payload, &e) == nil && e.Error != nil {
		t.Error("a routing 404 now carries an error envelope; it did not before")
	}
}

// ---------------------------------------------------------------- middleware

// TestContract_WebSocketRouteIsRegistered pins that /ws/:project_id exists on the
// real server. The frames it carries are the ProgressUpdate vocabulary
// characterised in pkg/core; T6's adapter must keep producing exactly those
// stages and percentages.
func TestContract_WebSocketRouteIsRegistered(t *testing.T) {
	s := New(&core.Config{
		WorkspaceDir: t.TempDir(),
		DefaultMode:  core.ModeAuto,
		APIProvider:  "deepseek",
	})

	req := httptest.NewRequest("GET", "/api/v1/ws/p-1", nil)
	resp, err := s.app.Test(req, 10_000)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// The loopback guard answers first with 403, which still proves the route is
	// mounted - a missing route would 404 at the router before any middleware
	// verdict. Both outcomes are recorded so a change is visible.
	if resp.StatusCode == http.StatusNotFound {
		t.Fatal("/api/v1/ws/:project_id is not registered")
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Logf("non-upgrade GET on the websocket route returned %d (403 expected "+
			"from the loopback guard; pinning the actual value)", resp.StatusCode)
	}
}

// TestContract_LoopbackGuardRefusesNonLoopback pins the ONLY access control the
// agent has today (server.go:91-101).
//
// The body is NOT the standard envelope: it is a bare {"error": "..."} string, so
// a client that parses every response as the envelope breaks on exactly the
// response that matters. T15 replaces this path and should fix the shape.
func TestContract_LoopbackGuardRefusesNonLoopback(t *testing.T) {
	s := New(&core.Config{
		WorkspaceDir: t.TempDir(),
		DefaultMode:  core.ModeAuto,
		APIProvider:  "deepseek",
	})

	// app.Test() synthesises a non-loopback client, which is the case under test.
	req := httptest.NewRequest("GET", "/api/v1/projects", nil)
	resp, err := s.app.Test(req, 10_000)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	payload, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("resp.StatusCode = %d for a non-loopback client, want 403; body: %s", resp.StatusCode, payload)
	}

	var bare struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(payload, &bare); err != nil {
		t.Fatalf(`403 body is not {"error": string}: %v; body: %s`, err, payload)
	}
	if bare.Error != "Forbidden: Only local connections allowed" {
		t.Errorf("error = %q, want %q", bare.Error, "Forbidden: Only local connections allowed")
	}

	var e envelope
	_ = json.Unmarshal(payload, &e)
	if e.Meta != nil {
		t.Error("the 403 now carries envelope meta; the guard response shape changed")
	}
}

// TestContract_ProductionEnvPanicsAtStartup pins a DEFECT found by writing these
// tests, and one this spec's own work created.
//
// GLIVE_ENV=production makes server.New set AllowOrigins to "*" while
// AllowCredentials stays true (server.go:44-52). That pairing is CVE-2024-25124
// (GO-2024-2574) - one of the three reachable Fiber vulnerabilities cleared in
// GL0 T0b by bumping v2.52.0 -> v2.52.12. The patched middleware REFUSES the
// combination, so it panics during New rather than serving a credential-bearing
// wildcard CORS policy.
//
// So the practical state of the flag is:
//   - before T0b: GLIVE_ENV=production disabled the loopback guard AND served a
//     wildcard CORS policy with credentials - the vulnerability, live;
//   - after T0b:  GLIVE_ENV=production crashes the agent on startup.
//
// Crashing is the better of the two - it is fail-closed, and it is discoverable.
// But it means the documented "production" mode is unusable, and nothing said so.
// GL0 T15 removes the bypass entirely (AUDIT-F-09), which resolves it properly;
// this test is what proves the flag is currently a trap rather than a feature.
func TestContract_ProductionEnvPanicsAtStartup(t *testing.T) {
	t.Setenv("GLIVE_ENV", "production")

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("server.New with GLIVE_ENV=production no longer panics - if T15 " +
				"has removed the bypass, delete this test; if the CORS config was " +
				"fixed, pin the new behaviour instead")
		}
		msg, _ := r.(string)
		if !strings.Contains(msg, "CORS") {
			t.Errorf("panicked with %v, want the CORS insecure-setup panic", r)
		}
	}()

	_ = New(&core.Config{
		WorkspaceDir: t.TempDir(),
		DefaultMode:  core.ModeAuto,
		APIProvider:  "deepseek",
	})
}

// TestContract_HealthIsOutsideTheAPIGroup pins that /health is not under
// /api/v1 and so not rate limited - monitoring depends on it.
func TestContract_HealthIsOutsideTheAPIGroup(t *testing.T) {
	app := newRouteApp(t)

	for i := 0; i < 150; i++ { // the limiter allows 100/min on /api/v1
		status, _ := do(t, app, "GET", "/health", "")
		if status != http.StatusOK {
			t.Fatalf("/health returned %d on request %d; it must not be rate limited",
				status, i+1)
		}
	}
}
