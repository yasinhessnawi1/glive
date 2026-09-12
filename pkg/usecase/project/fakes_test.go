package project_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/glive/domain/entities"
	"github.com/glive/domain/repository"
	"github.com/glive/domain/values"
	"github.com/glive/infrastructure/analyzer"
	"github.com/glive/infrastructure/container"
	"github.com/glive/infrastructure/executor"
	"github.com/glive/infrastructure/git"
	"github.com/glive/infrastructure/scanner"
)

// Fakes for characterising SetupProjectUseCase.Execute. They are supplied to the
// real container through the option seam added for this purpose, so Execute runs
// its real orchestration while every collaborator is inert - no network, no AI,
// no commands run.
//
// recorder is shared by all of them so the ORDER of the steps can be asserted,
// not just the fact that each happened. That ordering is the part the T6 collapse
// is most likely to change silently.

// recorder collects an ordered trace of collaborator calls. Safe for concurrent
// use because Execute runs the AI analysis on its own goroutine.
type recorder struct {
	mu     sync.Mutex
	events []string
}

func (r *recorder) record(event string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

func (r *recorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.events))
	copy(out, r.events)
	return out
}

// indexOf returns the position of the first event equal to name, or -1.
func (r *recorder) indexOf(name string) int {
	for i, e := range r.snapshot() {
		if e == name {
			return i
		}
	}
	return -1
}

// ---------------------------------------------------------------- git

type fakeGitClient struct {
	rec       *recorder
	localPath string
	cloneErr  error

	// conflictOnce makes the FIRST Clone return a *git.ConflictError, so the
	// conflict-resolution branch at setup_project.go:210 can be driven. The retry
	// after Delete/Rename then succeeds.
	conflictOnce bool
	clones       int
	deleted      bool
	renamed      bool
}

func (f *fakeGitClient) Clone(workspaceDir string, progress io.Writer) error {
	f.rec.record("clone")
	f.clones++
	if f.conflictOnce && f.clones == 1 {
		return &git.ConflictError{
			Path:      f.localPath,
			IsGitRepo: false,
			Message:   "directory already exists but is not a git repository",
		}
	}
	if f.cloneErr != nil {
		return f.cloneErr
	}
	// Materialise the directory Execute goes on to analyse and scan.
	return os.MkdirAll(f.localPath, 0o750)
}

func (f *fakeGitClient) CloneWithAuth(workspaceDir, username, token string, progress io.Writer) error {
	return f.Clone(workspaceDir, progress)
}
func (f *fakeGitClient) GetLocalPath() string  { return f.localPath }
func (f *fakeGitClient) IsCloned() bool        { return false }
func (f *fakeGitClient) DeleteExisting() error { f.deleted = true; return nil }
func (f *fakeGitClient) RenameExisting() (string, error) {
	f.renamed = true
	return f.localPath + ".backup", nil
}

type fakeGitFactory struct{ client *fakeGitClient }

func (f *fakeGitFactory) Create(githubURL string) (container.GitClient, error) { return f.client, nil }

// ---------------------------------------------------------------- ai

type fakeAIClient struct {
	rec      *recorder
	response string
	err      error

	mu      sync.Mutex
	autoFix int // AutoFixError call count; the loop itself is characterised in
	// pkg/infrastructure/executor, where it lives
}

func (f *fakeAIClient) AnalyzeProject(ctx context.Context, projectPath, readme string, files []string) (string, error) {
	f.rec.record("ai.AnalyzeProject")
	return f.response, f.err
}

func (f *fakeAIClient) DebugError(command, output, errorMsg string) (string, error) {
	f.rec.record("ai.DebugError")
	return "", nil
}

func (f *fakeAIClient) AutoFixError(command, output, errorMsg, workingDir string) (string, string, bool, error) {
	f.mu.Lock()
	f.autoFix++
	f.mu.Unlock()
	f.rec.record("ai.AutoFixError")
	return "", "", false, nil
}

type fakeAIFactory struct{ client container.AIClient }

func (f *fakeAIFactory) Create() container.AIClient { return f.client }

// ---------------------------------------------------------------- analyzer

type fakeAnalyzer struct {
	rec    *recorder
	result *analyzer.AnalysisResult
	err    error
}

func (f *fakeAnalyzer) Analyze() (*analyzer.AnalysisResult, error) {
	f.rec.record("analyze")
	return f.result, f.err
}

type fakeAnalyzerFactory struct{ a *fakeAnalyzer }

func (f *fakeAnalyzerFactory) Create(projectPath string) container.Analyzer { return f.a }

// ---------------------------------------------------------------- scanner

type fakeScanner struct {
	rec    *recorder
	result *scanner.ScanResult
	err    error
}

func (f *fakeScanner) Scan() (*scanner.ScanResult, error) {
	f.rec.record("scan")
	return f.result, f.err
}

type fakeScannerFactory struct{ s *fakeScanner }

func (f *fakeScannerFactory) Create(projectPath string) container.Scanner { return f.s }

// ---------------------------------------------------------------- executor

type fakeExecutor struct {
	rec      *recorder
	executed []string
	failWith error
	mu       sync.Mutex
}

func (f *fakeExecutor) Execute(ctx context.Context, cmd *executor.Command, out executor.OutputHandler) error {
	f.mu.Lock()
	f.executed = append(f.executed, cmd.Command)
	f.mu.Unlock()
	f.rec.record("execute:" + cmd.Command)
	if f.failWith != nil {
		cmd.Status = executor.CommandFailed
		cmd.ExitCode = 1
		return f.failWith
	}
	cmd.Status = executor.CommandCompleted
	return nil
}

func (f *fakeExecutor) ExecuteMultiple(ctx context.Context, cmds []*executor.Command, out executor.OutputHandler) error {
	for _, c := range cmds {
		if err := f.Execute(ctx, c, out); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeExecutor) ran() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.executed))
	copy(out, f.executed)
	return out
}

type fakeExecutorFactory struct{ e *fakeExecutor }

func (f *fakeExecutorFactory) Create(workingDir string, mode executor.ExecutionMode) container.Executor {
	return f.e
}

// ---------------------------------------------------------------- repository

// memProjectRepo is an in-memory ProjectRepository. It also records every Save so
// the status transitions Execute drives can be asserted.
type memProjectRepo struct {
	mu       sync.Mutex
	projects map[string]*entities.Project
	statuses []entities.ProjectStatus
	saveErr  error

	// existing is returned by FindByURL, so the "project already exists" branch
	// at setup_project.go:123 can be driven. deleted records whether the --force
	// path removed it.
	existing *entities.Project
	deleted  bool
}

func newMemProjectRepo() *memProjectRepo {
	return &memProjectRepo{projects: map[string]*entities.Project{}}
}

func (m *memProjectRepo) Save(ctx context.Context, p *entities.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.saveErr != nil {
		return m.saveErr
	}
	m.projects[p.ID().Value()] = p
	m.statuses = append(m.statuses, p.Status())
	return nil
}

func (m *memProjectRepo) FindByID(ctx context.Context, id *values.ProjectID) (*entities.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.projects[id.Value()]; ok {
		return p, nil
	}
	return nil, repository.ErrProjectNotFound
}

func (m *memProjectRepo) FindByURL(ctx context.Context, url *values.URL) (*entities.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.existing != nil {
		return m.existing, nil
	}
	return nil, repository.ErrProjectNotFound
}

// seedExisting installs a project that FindByURL will return.
func (m *memProjectRepo) seedExisting(t interface{ Fatalf(string, ...any) }, rawURL string) {
	u, err := values.NewURL(rawURL)
	if err != nil {
		t.Fatalf("seedExisting: %v", err)
	}
	p, err := entities.NewProject(values.GenerateProjectID(), u, "hello-world")
	if err != nil {
		t.Fatalf("seedExisting: %v", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.existing = p
}

func (m *memProjectRepo) List(ctx context.Context, f repository.ProjectFilter) ([]*entities.Project, error) {
	return nil, nil
}

func (m *memProjectRepo) Delete(ctx context.Context, id *values.ProjectID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleted = true
	m.existing = nil
	return nil
}

func (m *memProjectRepo) statusTrail() []entities.ProjectStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]entities.ProjectStatus, len(m.statuses))
	copy(out, m.statuses)
	return out
}

// ---------------------------------------------------------------- logger

type nullLogger struct{}

func (nullLogger) Info(string)         {}
func (nullLogger) Error(string, error) {}
func (nullLogger) Debug(string)        {}

// ---------------------------------------------------------------- harness

// harness bundles a container wired entirely from fakes with the recorder they
// share, so a test can drive Execute and then assert on the trace.
type harness struct {
	container *container.Container
	repo      *memProjectRepo
	rec       *recorder
	ai        *fakeAIClient
	exec      *fakeExecutor
	git       *fakeGitClient
	localPath string
}

// harnessOptions tunes the fakes for a particular case.
type harnessOptions struct {
	apiKey     string // "" means: no AI client, so Execute skips step 2
	analysis   *analyzer.AnalysisResult
	analyzeErr error
	scanResult *scanner.ScanResult
	scanErr    error
	aiResponse string
	aiErr      error
	cloneErr   error
	executeErr error
}

func newHarness(t interface {
	Fatalf(string, ...any)
	TempDir() string
	Helper()
}, opts harnessOptions) *harness {
	t.Helper()

	workspace := t.TempDir()
	local := filepath.Join(workspace, "fixture-repo")
	rec := &recorder{}

	if opts.analysis == nil {
		opts.analysis = &analyzer.AnalysisResult{ProjectType: "nodejs"}
	}
	if opts.scanResult == nil {
		opts.scanResult = &scanner.ScanResult{SuspiciousReasons: []string{}, Findings: []scanner.Finding{}}
	}

	gitClient := &fakeGitClient{rec: rec, localPath: local, cloneErr: opts.cloneErr}
	aiClient := &fakeAIClient{rec: rec, response: opts.aiResponse, err: opts.aiErr}
	execFake := &fakeExecutor{rec: rec, failWith: opts.executeErr}
	repo := newMemProjectRepo()

	cfg := &container.Config{
		APIKey:       opts.apiKey,
		APIProvider:  "deepseek",
		WorkspaceDir: workspace,
		DefaultMode:  executor.ModeAuto,
	}

	containerOpts := []container.Option{
		container.WithGitClientFactory(&fakeGitFactory{client: gitClient}),
		container.WithExecutorFactory(&fakeExecutorFactory{e: execFake}),
		container.WithAnalyzerFactory(&fakeAnalyzerFactory{a: &fakeAnalyzer{rec: rec, result: opts.analysis, err: opts.analyzeErr}}),
		container.WithScannerFactory(&fakeScannerFactory{s: &fakeScanner{rec: rec, result: opts.scanResult, err: opts.scanErr}}),
		container.WithProjectRepository(repo),
	}

	// An empty API key models "no AI configured": the real aiClientFactoryImpl is
	// left in place only when a key is set, so that the hasAI branch at
	// setup_project.go:171 is exercised both ways.
	if opts.apiKey != "" {
		containerOpts = append(containerOpts, container.WithAIClientFactory(&fakeAIFactory{client: aiClient}))
	} else {
		containerOpts = append(containerOpts, container.WithAIClientFactory(&fakeAIFactory{client: nil}))
	}

	cont, err := container.NewContainer(cfg, nullLogger{}, containerOpts...)
	if err != nil {
		t.Fatalf("NewContainer: %v", err)
	}

	return &harness{
		container: cont, repo: repo, rec: rec,
		ai: aiClient, exec: execFake, git: gitClient, localPath: local,
	}
}

// cmd builds an analyzer command for the fake analysis result.
func cmd(id, command string) analyzer.Command {
	return analyzer.Command{
		ID:          id,
		Description: fmt.Sprintf("run %s", command),
		Command:     command,
		Stage:       "setup",
		Required:    true,
	}
}
