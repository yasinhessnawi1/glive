package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/glive/core/analyzer"
	"github.com/glive/core/executor"
	"github.com/glive/core/repo"
	"github.com/glive/core/scanner"
	"github.com/glive/core/state"
	"github.com/glive/core/types"
	"github.com/glive/infrastructure/ai"
)

// Orchestrator coordinates the entire project execution flow
type Orchestrator struct {
	config           *Config
	stateManager     *state.Manager
	aiClient         *ai.Client
	output           io.Writer
	runningProjects  map[string]context.CancelFunc
	projectPorts     map[string]int // Track allocated ports per project
	nextPort         int            // Next available port
	mu               sync.Mutex
	currentCallback  ProgressCallback // Current progress callback for streaming logs
	currentProjectID string           // Current project ID for log streaming
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(cfg *Config, output io.Writer) (*Orchestrator, error) {
	if output == nil {
		output = os.Stdout
	}

	// Create state manager
	stateDir := filepath.Join(cfg.WorkspaceDir, ".glive")
	stateMgr, err := state.New(stateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	// Create AI client
	var aiClient *ai.Client
	if cfg.APIKey != "" {
		aiClient = ai.NewClient(cfg.APIKey, cfg.APIProvider, cfg.APIEndpoint)
	}

	return &Orchestrator{
		config:          cfg,
		stateManager:    stateMgr,
		aiClient:        aiClient,
		output:          output,
		runningProjects: make(map[string]context.CancelFunc),
		projectPorts:    make(map[string]int),
		nextPort:        8081, // Start from 8081 (8080 is for the agent)
	}, nil
}

// ProgressCallback is called for progress updates
type ProgressCallback func(update ProgressUpdate)

// RunProject orchestrates the entire process of running a GitHub project
func (o *Orchestrator) RunProject(ctx context.Context, projectID, githubURL string, mode ExecutionMode, forceExecution bool, callback ProgressCallback) (*Project, error) {
	if projectID == "" {
		projectID = generateProjectID()
	}

	// Set callback for log streaming
	o.mu.Lock()
	o.currentCallback = callback
	o.currentProjectID = projectID
	o.mu.Unlock()

	// Clear callback when done
	defer func() {
		o.mu.Lock()
		o.currentCallback = nil
		o.currentProjectID = ""
		o.mu.Unlock()
	}()

	o.log("🚀 GLive - GitHub to Live\n")
	o.log("=" + repeatString("=", 50) + "\n\n")

	// Helper function to both log to console AND stream through WebSocket
	logAndStream := func(message string) {
		o.log(message)
		o.sendProgress(callback, projectID, "running", message, 0)
	}

	// Step 1: Parse GitHub URL
	logAndStream("📋 Step 1: Parsing GitHub URL...\n")
	o.sendProgress(callback, projectID, "parsing", "Parsing GitHub URL", 10)

	repository, err := repo.ParseGitHubURL(githubURL)
	if err != nil {
		return nil, fmt.Errorf("invalid GitHub URL: %w", err)
	}

	logAndStream(fmt.Sprintf("   ✓ Repository: %s/%s\n", repository.Owner, repository.Name))
	logAndStream(fmt.Sprintf("   ✓ URL: %s\n\n", repository.URL))

	// Create project
	project := &Project{
		ID:        projectID,
		Name:      repository.Name,
		GitHubURL: githubURL,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save initial state
	if err := o.stateManager.SaveProject(project); err != nil {
		return nil, fmt.Errorf("failed to save project: %w", err)
	}

	// Step 2: Clone repository
	o.log("📦 Step 2: Cloning repository...\n")
	o.sendProgress(callback, projectID, "cloning", "Cloning repository", 20)

	project.Status = StatusCloning
	o.stateManager.SaveProject(project)

	// If force execution is enabled, remove existing directory first
	if forceExecution {
		localPath := filepath.Join(o.config.WorkspaceDir, repository.Name)
		if _, err := os.Stat(localPath); err == nil {
			o.log("   ⚠️  Force execution enabled: Removing existing directory...\n")
			if err := os.RemoveAll(localPath); err != nil {
				o.log(fmt.Sprintf("   ⚠️  Failed to remove existing directory: %v\n", err))
			}
		}
	}

	err = repository.Clone(o.config.WorkspaceDir, o.output)
	if err != nil {
		// If clone failed because it exists, and we didn't force, just continue
		if strings.Contains(err.Error(), "already exists") && !forceExecution {
			o.log("   ℹ️  Repository already exists, using existing files\n")
		} else {
			project.Status = StatusFailed
			o.stateManager.SaveProject(project)
			return nil, fmt.Errorf("failed to clone repository: %w", err)
		}
	}

	project.LocalPath = repository.GetLocalPath()
	o.log(fmt.Sprintf("   ✓ Cloned to: %s\n\n", project.LocalPath))

	// Step 3: Security scan
	o.log("🛡️  Step 3: Security scanning...\n")
	o.sendProgress(callback, projectID, "scanning", "Scanning for security issues", 30)

	secScanner := scanner.New(project.LocalPath)
	scanResult, err := secScanner.Scan()
	if err != nil {
		o.log(fmt.Sprintf("   ⚠️  Warning: Security scan failed: %v\n\n", err))
	} else {
		if scanResult.IsSuspicious {
			o.log("   ⚠️  WARNING: Potentially suspicious code detected!\n")
			for _, reason := range scanResult.SuspiciousReasons {
				o.log(fmt.Sprintf("      - %s\n", reason))
			}

			if mode != ModeAuto {
				o.log("\n   Do you want to continue? This is a security risk!\n")
				// In manual/assisted mode, user should confirm
				// For now, we'll continue with a warning
			}
		} else {
			o.log("   ✓ No security issues detected\n")
		}
		o.log(fmt.Sprintf("   ✓ Scanned %d files\n\n", len(scanResult.Findings)))
	}

	// Step 4: Analyze project
	o.log("🔍 Step 4: Analyzing project...\n")
	o.sendProgress(callback, projectID, "analyzing", "Analyzing project structure", 40)

	project.Status = StatusAnalyzing
	o.stateManager.SaveProject(project)

	projectAnalyzer := analyzer.New(project.LocalPath)
	analysis, err := projectAnalyzer.Analyze()
	if err != nil {
		return nil, fmt.Errorf("failed to analyze project: %w", err)
	}

	project.Type = analysis.ProjectType
	o.log(fmt.Sprintf("   ✓ Project type: %s\n", analysis.ProjectType))
	o.log(fmt.Sprintf("   ✓ Package managers: %v\n", analysis.PackageManagers))
	o.log(fmt.Sprintf("   ✓ Entry points: %v\n", analysis.EntryPoints))
	o.log(fmt.Sprintf("   ✓ Dependencies: %d\n\n", len(analysis.Dependencies)))

	// Step 5: AI-enhanced analysis (if API key is configured)
	if o.aiClient != nil {
		o.log("🤖 Step 5: AI-powered analysis...\n")
		o.sendProgress(callback, projectID, "ai_analysis", "Running AI analysis", 50)

		// Read README content
		readmeContent := o.readReadme(project.LocalPath)

		// Get file list
		fileList := o.getFileList(project.LocalPath)

		// Create a context with timeout for AI analysis (2 minutes)
		aiCtx, aiCancel := context.WithTimeout(ctx, 2*time.Minute)
		defer aiCancel()

		o.log("   ⏳ Analyzing project with AI (this may take up to 2 minutes)...\n")

		// Use AI to enhance the analysis
		aiResponse, err := o.aiClient.AnalyzeProject(aiCtx, project.LocalPath, readmeContent, fileList)
		if err != nil {
			// Check if it was a timeout
			if aiCtx.Err() == context.DeadlineExceeded {
				o.log("   ⚠️  AI analysis timed out after 2 minutes\n")
			} else {
				o.log(fmt.Sprintf("   ⚠️  AI analysis failed: %v\n", err))
			}
			o.log("   ℹ️  Falling back to basic analysis - the project will still work!\n\n")
		} else {
			o.log("   ✓ AI analysis completed\n")

			// Merge AI suggestions with basic analysis
			if err := o.mergeAIAnalysis(analysis, aiResponse); err != nil {
				o.log(fmt.Sprintf("   ⚠️  Failed to merge AI analysis: %v\n", err))
				o.log("   ℹ️  Using basic analysis instead\n")
			} else {
				o.log("   ✓ Enhanced with AI insights\n")
				if analysis.Description != "" {
					o.log(fmt.Sprintf("   ✓ Description: %s\n", analysis.Description))
				}
			}
			o.log("\n")
		}
	} else {
		o.log("⚠️  Step 5: Skipping AI analysis (no API key configured)\n")
		o.log("   ℹ️  Set API key to enable smart features: glive config set api-key YOUR_KEY\n\n")
	}

	// Step 6: Execute setup commands
	o.log("⚙️  Step 6: Setting up project...\n")
	o.sendProgress(callback, projectID, "installing", "Installing dependencies", 60)

	project.Status = StatusInstalling
	o.stateManager.SaveProject(project)

	if len(analysis.Commands) > 0 {
		exec := executor.New(project.LocalPath, mode, nil) // TODO: Create adapter for infrastructure AI client

		for i, cmd := range analysis.Commands {
			percentage := 60 + (i * 30 / len(analysis.Commands))
			o.sendProgress(callback, projectID, "executing", cmd.Description, percentage)

			o.log(fmt.Sprintf("   [%d/%d] %s\n", i+1, len(analysis.Commands), cmd.Description))
			o.log(fmt.Sprintf("   $ %s\n", cmd.Command))

			err := exec.Execute(ctx, &cmd, func(line string) {
				o.log(fmt.Sprintf("      %s\n", line))
			})

			if err != nil {
				if cmd.Required {
					project.Status = StatusFailed
					o.stateManager.SaveProject(project)
					return nil, fmt.Errorf("required command failed: %w", err)
				}
				o.log(fmt.Sprintf("   ⚠️  Command failed (optional): %v\n", err))
			} else {
				o.log("   ✓ Command completed successfully\n")
			}
			o.log("\n")
		}
	} else {
		o.log("   ℹ️  No setup commands detected\n\n")
	}

	// Step 7: Complete
	o.log("✅ Step 7: Project ready!\n")
	o.sendProgress(callback, projectID, "ready", "Project is ready", 100)

	project.Status = StatusReady
	project.UpdatedAt = time.Now()
	o.stateManager.SaveProject(project)

	o.log("=" + repeatString("=", 50) + "\n")
	o.log(fmt.Sprintf("🎉 Project '%s' is ready to use!\n", project.Name))
	o.log(fmt.Sprintf("📁 Location: %s\n\n", project.LocalPath))

	return project, nil
}

// StartProject starts a project's application
func (o *Orchestrator) StartProject(ctx context.Context, projectID string, callback ProgressCallback) error {
	project, err := o.stateManager.LoadProject(projectID)
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}

	// Set callback for log streaming
	o.mu.Lock()
	o.currentCallback = callback
	o.currentProjectID = projectID

	// Check if already running
	if _, exists := o.runningProjects[projectID]; exists {
		o.mu.Unlock()
		return fmt.Errorf("project is already running")
	}
	o.mu.Unlock()

	// Find run command
	// We need to re-analyze or check stored commands.
	// The project struct doesn't store commands directly, but AnalysisResult does.
	// We should probably store commands in the project struct or re-analyze.
	// For now, let's re-analyze quickly since we have the path.
	// Ideally, we should have stored the analysis result.

	// Optimization: Check if we can just run the "run" command if we knew it.
	// Since we don't persist commands in Project struct (yet), let's re-analyze.
	// It's fast enough for now.

	projectAnalyzer := analyzer.New(project.LocalPath)
	analysis, err := projectAnalyzer.Analyze()
	if err != nil {
		return fmt.Errorf("failed to analyze project: %w", err)
	}

	var runCmd *types.Command

	// Check if dependencies are installed (auto-recovery for Node.js)
	if analysis.ProjectType == ProjectTypeNodeJS {
		nodeModulesPath := filepath.Join(project.LocalPath, "node_modules")
		if _, err := os.Stat(nodeModulesPath); os.IsNotExist(err) {
			o.log("   ⚠️  Node modules not found, installing dependencies...\n")
			o.sendProgress(callback, projectID, "installing", "Installing dependencies (auto-recovery)", 0)

			exec := executor.New(project.LocalPath, ModeAuto, nil) // TODO: Create adapter
			for _, cmd := range analysis.Commands {
				if cmd.Stage == "setup" {
					o.log(fmt.Sprintf("   $ %s\n", cmd.Command))
					err := exec.Execute(ctx, &cmd, func(line string) {
						o.log(fmt.Sprintf("      %s\n", line))
					})
					if err != nil {
						return fmt.Errorf("failed to install dependencies: %w", err)
					}
				}
			}
			o.log("   ✓ Dependencies installed\n")
		}
	}

	for _, cmd := range analysis.Commands {
		if cmd.Stage == "run" {
			runCmd = &cmd
			break
		}
	}

	if runCmd == nil {
		return fmt.Errorf("no start command found for project")
	}

	// Allocate a unique port for this project
	o.mu.Lock()
	var port int
	if existingPort, exists := o.projectPorts[projectID]; exists {
		// Reuse the same port if project was stopped and restarted
		port = existingPort
	} else {
		// Allocate next available port
		port = o.nextPort
		o.projectPorts[projectID] = port
		o.nextPort++ // Increment for next project
	}
	o.mu.Unlock()

	// Inject environment variables with allocated port
	if runCmd.Env == nil {
		runCmd.Env = make(map[string]string)
	}
	runCmd.Env["GLIVE_AGENT_PORT"] = fmt.Sprintf("%d", port)
	runCmd.Env["PORT"] = fmt.Sprintf("%d", port) // Common standard

	o.log(fmt.Sprintf("   📍 Allocated port %d for this project\n", port))

	// Create cancellable context for the running process
	runCtx, cancel := context.WithCancel(context.Background())

	o.mu.Lock()
	o.runningProjects[projectID] = cancel
	o.mu.Unlock()

	project.Status = StatusRunning
	o.stateManager.SaveProject(project)

	o.log(fmt.Sprintf("🚀 Starting project '%s'...\n", project.Name))
	o.sendProgress(callback, projectID, "starting", "Starting application", 0)

	// Run in background
	go func() {
		defer cancel() // Ensure context is cancelled on exit

		// Clear callback when goroutine exits
		defer func() {
			o.mu.Lock()
			o.currentCallback = nil
			o.currentProjectID = ""
			o.mu.Unlock()
		}()

		exec := executor.New(project.LocalPath, ModeAuto, nil) // TODO: Create adapter

		o.log(fmt.Sprintf("   $ %s\n", runCmd.Command))

		err := exec.Execute(runCtx, runCmd, func(line string) {
			o.log(fmt.Sprintf("      %s\n", line))

			// Stream logs to callback
			// We use a special stage "log" or just reuse "running" with the message
			o.sendProgress(callback, projectID, "running", line, 0)
		})

		o.mu.Lock()
		delete(o.runningProjects, projectID)
		o.mu.Unlock()

		if err != nil {
			// If cancelled, it's not an error
			if runCtx.Err() == context.Canceled {
				o.log("   🛑 Project stopped\n")
				project.Status = StatusStopped // or ready?
			} else {
				o.log(fmt.Sprintf("   ⚠️  Project crashed: %v\n", err))
				project.Status = StatusFailed
			}
		} else {
			o.log("   ✅ Project finished successfully\n")
			project.Status = StatusReady
		}
		o.stateManager.SaveProject(project)

		o.sendProgress(callback, projectID, "stopped", "Project stopped", 100)
	}()

	return nil
}

// StopProject stops a running project
func (o *Orchestrator) StopProject(projectID string) error {
	o.mu.Lock()
	cancel, exists := o.runningProjects[projectID]
	o.mu.Unlock()

	if !exists {
		return fmt.Errorf("project is not running")
	}

	cancel()
	return nil
}

// CleanupProject cleans up a project (stops and deletes)
func (o *Orchestrator) CleanupProject(projectID string) error {
	// Stop if running
	o.StopProject(projectID) // Ignore error if not running

	// Delete from state
	if err := o.stateManager.DeleteProject(projectID); err != nil {
		return fmt.Errorf("failed to delete project state: %w", err)
	}

	// Free up the allocated port
	o.mu.Lock()
	delete(o.projectPorts, projectID)
	o.mu.Unlock()

	// TODO: Delete files from disk?
	// For now, we keep files.

	return nil
}

// log writes to the output writer and streams via WebSocket if callback is set
func (o *Orchestrator) log(message string) {
	fmt.Fprint(o.output, message)

	// Also send through WebSocket callback if available
	if o.currentCallback != nil && o.currentProjectID != "" {
		// Send as a "log" stage progress update
		o.currentCallback(ProgressUpdate{
			ProjectID:  o.currentProjectID,
			Stage:      "log",
			Message:    message,
			Percentage: 0,
			Timestamp:  time.Now(),
		})
	}
}

// sendProgress sends a progress update
func (o *Orchestrator) sendProgress(callback ProgressCallback, projectID, stage, message string, percentage int) {
	if callback != nil {
		callback(ProgressUpdate{
			ProjectID:  projectID,
			Stage:      stage,
			Message:    message,
			Percentage: percentage,
			Timestamp:  time.Now(),
		})
	}
}

// readReadme finds and reads the README file
func (o *Orchestrator) readReadme(projectPath string) string {
	readmeFiles := []string{"README.md", "README.MD", "readme.md", "README.txt", "README", "Readme.md"}

	for _, readmeFile := range readmeFiles {
		path := filepath.Join(projectPath, readmeFile)
		if data, err := os.ReadFile(path); err == nil {
			return string(data)
		}
	}

	return ""
}

// getFileList returns a list of files in the project (limited to important files)
func (o *Orchestrator) getFileList(projectPath string) []string {
	var files []string
	maxFiles := 100

	// Walk the directory tree
	filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip hidden directories and node_modules, etc.
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "__pycache__" ||
				name == "venv" || name == ".venv" || name == "target" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(projectPath, path)
		if err != nil {
			return nil
		}

		files = append(files, relPath)

		// Limit file list to avoid too large AI requests
		if len(files) >= maxFiles {
			return filepath.SkipAll
		}

		return nil
	})

	return files
}

// mergeAIAnalysis merges AI analysis results with the basic analysis
func (o *Orchestrator) mergeAIAnalysis(analysis *AnalysisResult, aiResponse string) error {
	// Parse the AI response JSON
	var aiResult struct {
		ProjectType        string    `json:"project_type"`
		DetectedLanguages  []string  `json:"detected_languages"`
		PackageManagers    []string  `json:"package_managers"`
		EntryPoints        []string  `json:"entry_points"`
		SystemRequirements []string  `json:"system_requirements"`
		Commands           []Command `json:"commands"`
		IsSuspicious       bool      `json:"is_suspicious"`
		SuspiciousReasons  []string  `json:"suspicious_reasons"`
		EstimatedSize      string    `json:"estimated_size"`
		Description        string    `json:"description"`
	}

	// Try to unmarshal the AI response
	if err := json.Unmarshal([]byte(aiResponse), &aiResult); err != nil {
		// If JSON parsing fails, try to extract JSON from the response
		// Sometimes AI adds extra text around the JSON
		start := strings.Index(aiResponse, "{")
		end := strings.LastIndex(aiResponse, "}")
		if start >= 0 && end > start {
			jsonStr := aiResponse[start : end+1]
			if err := json.Unmarshal([]byte(jsonStr), &aiResult); err != nil {
				return fmt.Errorf("failed to parse AI response: %w", err)
			}
		} else {
			return fmt.Errorf("no valid JSON found in AI response")
		}
	}

	// Merge the results - AI takes precedence for description and some fields
	if aiResult.Description != "" {
		analysis.Description = aiResult.Description
	}

	// Add AI-detected languages if not already present
	for _, lang := range aiResult.DetectedLanguages {
		found := false
		for _, existing := range analysis.DetectedLanguages {
			if existing == lang {
				found = true
				break
			}
		}
		if !found {
			analysis.DetectedLanguages = append(analysis.DetectedLanguages, lang)
		}
	}

	// Add AI system requirements
	if len(aiResult.SystemRequirements) > 0 {
		analysis.SystemRequirements = aiResult.SystemRequirements
	}

	// If AI provided better entry points, use them
	if len(aiResult.EntryPoints) > 0 {
		analysis.EntryPoints = aiResult.EntryPoints
	}

	// Merge commands - AI commands can enhance or replace basic ones
	if len(aiResult.Commands) > 0 {
		// For now, prefer AI commands as they're more context-aware
		analysis.Commands = aiResult.Commands
	}

	// Update estimated size if provided
	if aiResult.EstimatedSize != "" && aiResult.EstimatedSize != "unknown" {
		analysis.EstimatedSize = aiResult.EstimatedSize
	}

	// Add security insights from AI
	if aiResult.IsSuspicious && len(aiResult.SuspiciousReasons) > 0 {
		analysis.IsSuspicious = true
		analysis.SuspiciousReasons = append(analysis.SuspiciousReasons, aiResult.SuspiciousReasons...)
	}

	return nil
}

// Helper functions
func generateProjectID() string {
	return fmt.Sprintf("project-%d", time.Now().UnixNano())
}

func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
