package project

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/glive/domain/entities"
	"github.com/glive/domain/repository"
	"github.com/glive/domain/values"
	"github.com/glive/infrastructure/analyzer"
	"github.com/glive/infrastructure/container"
	"github.com/glive/infrastructure/executor"
	"github.com/glive/infrastructure/git"
	"github.com/glive/usecase/execution"
)

// SetupProjectInput represents the input for setting up a project
type SetupProjectInput struct {
	GitHubURL    string
	Mode         executor.ExecutionMode
	WorkspaceDir string
	Force        bool // Force re-clone even if project exists

	// AI-First options
	AIMode       string  // ai-first, traditional, hybrid
	AIConfidence float64 // Confidence threshold
	NoFallback   bool    // Disable fallback
	NoAIRecovery bool    // Disable AI recovery
}

// SetupProjectOutput represents the output of setting up a project
type SetupProjectOutput struct {
	Project  *entities.Project
	Commands []*entities.Command
	Warnings []string
}

// ConflictAction represents user's choice for handling directory conflicts
type ConflictAction int

const (
	ConflictActionNone ConflictAction = iota
	ConflictActionDelete
	ConflictActionRename
	ConflictActionCancel
)

// ConflictCallback is called when a directory conflict is detected
// It should return the user's chosen action
type ConflictCallback func(path string) ConflictAction

// SetupProjectUseCase handles the complete project setup workflow
type SetupProjectUseCase struct {
	projectRepo repository.ProjectRepository
	container   *container.Container
	output      io.Writer
	progressCB  ProgressCallback
	conflictCB  ConflictCallback
}

// ProgressCallback is called for progress updates
type ProgressCallback func(projectID string, stage string, message string, percentage int)

// NewSetupProjectUseCase creates a new SetupProjectUseCase
func NewSetupProjectUseCase(
	projectRepo repository.ProjectRepository,
	cont *container.Container,
	output io.Writer,
) *SetupProjectUseCase {
	if output == nil {
		output = os.Stdout
	}
	return &SetupProjectUseCase{
		projectRepo: projectRepo,
		container:   cont,
		output:      output,
	}
}

// SetProgressCallback sets the progress callback
func (uc *SetupProjectUseCase) SetProgressCallback(callback ProgressCallback) {
	uc.progressCB = callback
}

// SetConflictCallback sets the conflict resolution callback
func (uc *SetupProjectUseCase) SetConflictCallback(callback ConflictCallback) {
	uc.conflictCB = callback
}

// Execute executes the setup project use case
func (uc *SetupProjectUseCase) Execute(ctx context.Context, input SetupProjectInput) (*SetupProjectOutput, error) {
	// Check for cancellation before starting
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Step 1: Parse GitHub URL
	uc.log("📋 Step 1: Parsing GitHub URL...\n")
	uc.sendProgress("", "parsing", "Parsing GitHub URL", 10)

	url, err := values.NewURL(input.GitHubURL)
	if err != nil {
		return nil, fmt.Errorf("invalid GitHub URL: %w", err)
	}

	owner, name, err := url.ParseOwnerAndName()
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	uc.log(fmt.Sprintf("   ✓ Repository: %s/%s\n", owner, name))
	uc.log(fmt.Sprintf("   ✓ URL: %s\n\n", url.String()))

	// Check if project already exists
	existingProject, err := uc.projectRepo.FindByURL(ctx, url)
	if err == nil && existingProject != nil {
		if input.Force {
			uc.log("   ⚠️  Project exists, but --force flag set. Re-cloning...\n")
			// Delete existing project state and directory
			if existingProject.LocalPath() != nil {
				localPathStr := existingProject.LocalPath().String()
				uc.log(fmt.Sprintf("   🗑️  Removing existing directory: %s\n", localPathStr))
				if err := os.RemoveAll(localPathStr); err != nil {
					uc.log(fmt.Sprintf("   ⚠️  Warning: Failed to remove directory: %v\n", err))
				}
			}
			// Delete from repository
			if err := uc.projectRepo.Delete(ctx, existingProject.ID()); err != nil {
				uc.log(fmt.Sprintf("   ⚠️  Warning: Failed to delete project state: %v\n", err))
			}
			uc.log("\n")
		} else {
			uc.log("   ℹ️  Project already exists, using existing project...\n\n")
			// Send progress updates to mark all steps as complete
			uc.sendProgress(existingProject.ID().Value(), "parsing", "Using existing project", 100)
			uc.sendProgress(existingProject.ID().Value(), "ai_analysis", "Skipped", 100)
			uc.sendProgress(existingProject.ID().Value(), "cloning", "Already cloned", 100)
			uc.sendProgress(existingProject.ID().Value(), "scanning", "Previously scanned", 100)
			uc.sendProgress(existingProject.ID().Value(), "analyzing", "Previously analyzed", 100)
			uc.sendProgress(existingProject.ID().Value(), "installing", "Already installed", 100)
			uc.sendProgress(existingProject.ID().Value(), "ready", "Project ready", 100)
			return &SetupProjectOutput{
				Project:  existingProject,
				Commands: []*entities.Command{},
				Warnings: []string{"Project already exists"},
			}, nil
		}
	}

	// Create project entity
	projectID := values.GenerateProjectID()
	project, err := entities.NewProject(projectID, url, name)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	// Save initial state
	if err := uc.projectRepo.Save(ctx, project); err != nil {
		return nil, fmt.Errorf("failed to save project: %w", err)
	}

	// Step 2: Check AI availability (will start in background after clone)
	aiClient := uc.container.CreateAIClient()
	hasAI := aiClient != nil
	if hasAI {
		uc.log("🤖 Step 2: AI analysis will start after clone...\n")
		uc.sendProgress(projectID.Value(), "ai_analysis", "Waiting for clone", 5)
		uc.log("   ✓ API key configured\n\n")
	} else {
		uc.log("⚠️  Step 2: Skipping AI analysis (no API key configured)\n")
		uc.sendProgress(projectID.Value(), "ai_analysis", "Skipped - no API key", 100)
		uc.log("   ℹ️  Set API key to enable smart features: glive config set api-key YOUR_KEY\n\n")
	}

	// Step 3: Clone repository
	// Check for cancellation before cloning
	select {
	case <-ctx.Done():
		if err := project.SetStatus(entities.StatusStopped); err == nil {
			uc.projectRepo.Save(ctx, project)
		}
		return nil, ctx.Err()
	default:
	}

	uc.log("📦 Step 3: Cloning repository...\n")
	uc.sendProgress(projectID.Value(), "cloning", "Cloning repository", 20)

	if err := project.SetStatus(entities.StatusCloning); err != nil {
		return nil, fmt.Errorf("failed to set status: %w", err)
	}
	uc.projectRepo.Save(ctx, project)

	gitClient, err := uc.container.CreateGitClient(input.GitHubURL)
	if err != nil {
		if err := project.SetStatus(entities.StatusFailed); err == nil {
			uc.projectRepo.Save(ctx, project)
		}
		return nil, fmt.Errorf("failed to create git client: %w", err)
	}

	if err := gitClient.Clone(input.WorkspaceDir, uc.output); err != nil {
		// Check if this is a conflict error that can be resolved
		if conflictErr, ok := err.(interface{ Is(error) bool }); ok && conflictErr.Is(git.ErrDirectoryConflict) {
			if uc.conflictCB != nil {
				// Ask user for resolution
				uc.log("   ⚠️  Directory conflict detected!\n")
				action := uc.conflictCB(gitClient.GetLocalPath())

				switch action {
				case ConflictActionDelete:
					uc.log("   🗑️  Deleting existing directory...\n")
					if err := gitClient.DeleteExisting(); err != nil {
						return nil, fmt.Errorf("failed to delete existing directory: %w", err)
					}
					// Retry clone
					if err := gitClient.Clone(input.WorkspaceDir, uc.output); err != nil {
						if err := project.SetStatus(entities.StatusFailed); err == nil {
							uc.projectRepo.Save(ctx, project)
						}
						return nil, fmt.Errorf("failed to clone repository after delete: %w", err)
					}
				case ConflictActionRename:
					uc.log("   📁 Renaming existing directory...\n")
					backupPath, err := gitClient.RenameExisting()
					if err != nil {
						return nil, fmt.Errorf("failed to rename existing directory: %w", err)
					}
					uc.log(fmt.Sprintf("   ✓ Renamed to: %s\n", backupPath))
					// Retry clone
					if err := gitClient.Clone(input.WorkspaceDir, uc.output); err != nil {
						if err := project.SetStatus(entities.StatusFailed); err == nil {
							uc.projectRepo.Save(ctx, project)
						}
						return nil, fmt.Errorf("failed to clone repository after rename: %w", err)
					}
				case ConflictActionCancel:
					if err := project.SetStatus(entities.StatusStopped); err == nil {
						uc.projectRepo.Save(ctx, project)
					}
					return nil, fmt.Errorf("operation cancelled by user")
				default:
					if err := project.SetStatus(entities.StatusFailed); err == nil {
						uc.projectRepo.Save(ctx, project)
					}
					return nil, fmt.Errorf("failed to clone repository: %w", err)
				}
			} else {
				// No callback, just fail
				if err := project.SetStatus(entities.StatusFailed); err == nil {
					uc.projectRepo.Save(ctx, project)
				}
				return nil, fmt.Errorf("failed to clone repository: %w", err)
			}
		} else {
			if err := project.SetStatus(entities.StatusFailed); err == nil {
				uc.projectRepo.Save(ctx, project)
			}
			return nil, fmt.Errorf("failed to clone repository: %w", err)
		}
	}

	localPath, err := values.NewPath(gitClient.GetLocalPath())
	if err != nil {
		return nil, fmt.Errorf("invalid local path: %w", err)
	}

	if err := project.SetLocalPath(localPath); err != nil {
		return nil, fmt.Errorf("failed to set local path: %w", err)
	}

	uc.log(fmt.Sprintf("   ✓ Cloned to: %s\n\n", localPath.String()))
	uc.projectRepo.Save(ctx, project)

	// Start AI analysis in background (runs in parallel with security scan and basic analysis)
	type aiResult struct {
		response string
		err      error
	}
	aiResultChan := make(chan aiResult, 1)
	if hasAI {
		uc.log("🤖 Step 2: Starting AI analysis in background...\n")
		uc.sendProgress(projectID.Value(), "ai_analysis", "AI analyzing in background", 20)
		go func() {
			readmeContent := uc.readReadme(localPath.String())
			fileList := uc.getFileList(localPath.String())
			response, err := aiClient.AnalyzeProject(ctx, localPath.String(), readmeContent, fileList)
			aiResultChan <- aiResult{response: response, err: err}
		}()
	}

	// Step 4: Security scan
	// Check for cancellation before scanning
	select {
	case <-ctx.Done():
		if err := project.SetStatus(entities.StatusStopped); err == nil {
			uc.projectRepo.Save(ctx, project)
		}
		return nil, ctx.Err()
	default:
	}

	uc.log("🛡️  Step 4: Security scanning...\n")
	uc.sendProgress(projectID.Value(), "scanning", "Scanning for security issues", 40)

	scanner := uc.container.CreateScanner(localPath.String())
	scanResult, err := scanner.Scan()
	warnings := []string{}

	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Security scan failed: %v", err))
		uc.log(fmt.Sprintf("   ⚠️  Warning: Security scan failed: %v\n\n", err))
	} else {
		if scanResult.IsSuspicious {
			warnings = append(warnings, "Potentially suspicious code detected")
			uc.log("   ⚠️  WARNING: Potentially suspicious code detected!\n")
			for _, reason := range scanResult.SuspiciousReasons {
				uc.log(fmt.Sprintf("      - %s\n", reason))
				warnings = append(warnings, reason)
			}
			uc.log("\n")
		} else {
			uc.log("   ✓ No security issues detected\n")
		}
		uc.log(fmt.Sprintf("   ✓ Scanned %d files\n\n", len(scanResult.Findings)))
	}

	// Step 5: Analyze project (basic static analysis)
	// Check for cancellation before analyzing
	select {
	case <-ctx.Done():
		if err := project.SetStatus(entities.StatusStopped); err == nil {
			uc.projectRepo.Save(ctx, project)
		}
		return nil, ctx.Err()
	default:
	}

	uc.log("🔍 Step 5: Analyzing project...\n")
	uc.sendProgress(projectID.Value(), "analyzing", "Analyzing project structure", 50)

	if err := project.SetStatus(entities.StatusAnalyzing); err != nil {
		return nil, fmt.Errorf("failed to set status: %w", err)
	}
	uc.projectRepo.Save(ctx, project)

	projectAnalyzer := uc.container.CreateAnalyzer(localPath.String())
	analysisResult, err := projectAnalyzer.Analyze()
	if err != nil {
		if err := project.SetStatus(entities.StatusFailed); err == nil {
			uc.projectRepo.Save(ctx, project)
		}
		return nil, fmt.Errorf("failed to analyze project: %w", err)
	}

	project.SetType(entities.ProjectType(analysisResult.ProjectType))
	uc.log(fmt.Sprintf("   ✓ Project type: %s\n", analysisResult.ProjectType))
	uc.log(fmt.Sprintf("   ✓ Package managers: %v\n", analysisResult.PackageManagers))
	uc.log(fmt.Sprintf("   ✓ Entry points: %v\n", analysisResult.EntryPoints))
	uc.log(fmt.Sprintf("   ✓ Dependencies: %d\n\n", len(analysisResult.Dependencies)))

	// Wait for AI analysis result (started in background after clone)
	if hasAI {
		// Only log once, progress callback will handle UI updates
		uc.sendProgress(projectID.Value(), "ai_analysis", "Waiting for AI analysis result...", 60)

		// Wait for the background AI analysis to complete
		result := <-aiResultChan
		if result.err != nil {
			uc.log(fmt.Sprintf("   ⚠️  AI analysis failed: %v\n", result.err))
			uc.log("   ℹ️  Falling back to basic analysis - the project will still work!\n\n")
			uc.sendProgress(projectID.Value(), "ai_analysis", fmt.Sprintf("AI analysis failed: %v", result.err), 100)
		} else {
			uc.log("   ✓ AI analysis completed\n")
			if err := uc.mergeAIAnalysis(analysisResult, result.response); err != nil {
				uc.log(fmt.Sprintf("   ⚠️  Failed to merge AI analysis: %v\n", err))
				uc.log("   ℹ️  Using basic analysis instead\n")
				uc.sendProgress(projectID.Value(), "ai_analysis", "Failed to merge AI analysis", 100)
			} else {
				uc.log("   ✓ Enhanced with AI insights\n")
				if analysisResult.Description != "" {
					uc.log(fmt.Sprintf("   ✓ Description: %s\n", analysisResult.Description))
				}
				uc.sendProgress(projectID.Value(), "ai_analysis", "AI analysis complete", 100)
			}
			uc.log("\n")
		}
	}

	// Convert analysis result to domain entity
	domainAnalysis := uc.convertAnalysisToDomain(analysisResult, localPath)
	project.SetAnalysis(domainAnalysis)
	// Save progress after analysis is complete
	uc.projectRepo.Save(ctx, project)

	// Step 6: Execute setup commands
	// Check for cancellation before executing commands
	select {
	case <-ctx.Done():
		if err := project.SetStatus(entities.StatusStopped); err == nil {
			uc.projectRepo.Save(ctx, project)
		}
		return nil, ctx.Err()
	default:
	}

	uc.log("⚙️  Step 6: Setting up project...\n")
	uc.sendProgress(projectID.Value(), "installing", "Installing dependencies", 60)

	if err := project.SetStatus(entities.StatusInstalling); err != nil {
		return nil, fmt.Errorf("failed to set status: %w", err)
	}
	uc.projectRepo.Save(ctx, project)

	commands := domainAnalysis.Commands()
	if len(commands) > 0 {
		// Create output handler that writes to our io.Writer
		// Clear any spinner/progress before writing command output
		firstOutput := true
		outputHandler := func(line string) {
			// Clear spinner line before first output
			if firstOutput {
				// Use ANSI escape code to clear line and move to new line
				// This ensures command output doesn't interfere with progress indicators
				fmt.Fprint(uc.output, "\r\033[2K")
				firstOutput = false
			}
			// Ensure we're on a new line before writing command output
			// This prevents command output from interfering with progress indicators
			if !strings.HasSuffix(line, "\n") {
				fmt.Fprint(uc.output, line+"\n")
			} else {
				fmt.Fprint(uc.output, line)
			}
		}

		executeUC := execution.NewExecuteCommandsUseCase(uc.container, outputHandler)

		// Set AI client for auto-fix functionality
		if aiClient != nil {
			executeUC.SetAIClient(aiClient)
		}

		// Convert progress callback
		if uc.progressCB != nil {
			executeUC.SetProgressCallback(execution.ProgressCallback(uc.progressCB))
		}

		execCommands := make([]*executor.Command, 0, len(commands))
		for _, cmd := range commands {
			execCommands = append(execCommands, &executor.Command{
				ID:          cmd.ID(),
				Description: cmd.Description(),
				Command:     cmd.Command(),
				WorkingDir:  cmd.WorkingDir().String(),
				Stage:       string(cmd.Stage()),
				Required:    cmd.Required(),
				Status:      executor.CommandStatus(cmd.Status()),
			})
		}

		err := executeUC.Execute(ctx, execution.ExecuteCommandsInput{
			Commands:   execCommands,
			WorkingDir: localPath.String(),
			Mode:       input.Mode,
		})

		if err != nil {
			if err := project.SetStatus(entities.StatusFailed); err == nil {
				uc.projectRepo.Save(ctx, project)
			}
			return nil, fmt.Errorf("failed to execute commands: %w", err)
		}

		// Update command statuses from executor
		for i, execCmd := range execCommands {
			if i < len(commands) {
				cmd := commands[i]
				switch execCmd.Status {
				case executor.CommandCompleted:
					cmd.MarkCompleted(execCmd.Output)
				case executor.CommandFailed:
					cmd.MarkFailed(execCmd.Output, execCmd.Error, execCmd.ExitCode)
				case executor.CommandSkipped:
					cmd.MarkSkipped()
				}
			}
		}
	} else {
		uc.log("   ℹ️  No setup commands detected\n\n")
	}

	// Step 7: Complete
	uc.log("✅ Step 7: Project ready!\n")
	uc.sendProgress(projectID.Value(), "ready", "Project is ready", 100)

	if err := project.SetStatus(entities.StatusReady); err != nil {
		return nil, fmt.Errorf("failed to set status: %w", err)
	}
	uc.projectRepo.Save(ctx, project)

	// Generate Final Report
	uc.log("\n" + strings.Repeat("=", 60) + "\n")
	uc.log("🚀 FINAL PROJECT REPORT\n")
	uc.log(strings.Repeat("=", 60) + "\n\n")

	// 1. Objectives & Summary
	uc.log("📋 OBJECTIVES & SUMMARY\n")
	uc.log(fmt.Sprintf("   Target: %s\n", input.GitHubURL))
	uc.log(fmt.Sprintf("   Result: Successfully setup '%s'\n", project.Name()))
	if project.Analysis() != nil && project.Analysis().Description() != "" {
		uc.log(fmt.Sprintf("   About:  %s\n", project.Analysis().Description()))
	}
	uc.log("\n")

	// 2. What has been done
	uc.log("✅ WHAT HAS BEEN DONE\n")
	uc.log("   ✓ Cloned repository\n")
	uc.log("   ✓ Analyzed project structure\n")
	if project.Analysis() != nil {
		uc.log(fmt.Sprintf("   ✓ Identified %d dependencies\n", len(project.Analysis().Dependencies())))
	}
	if len(commands) > 0 {
		uc.log(fmt.Sprintf("   ✓ Executed %d setup commands\n", len(commands)))
	}
	uc.log("\n")

	// 3. Key Milestones (from AI)
	if project.Analysis() != nil && len(project.Analysis().KeyMilestones()) > 0 {
		uc.log("🚩 KEY MILESTONES ACHIEVED\n")
		for _, milestone := range project.Analysis().KeyMilestones() {
			uc.log(fmt.Sprintf("   ✓ %s\n", milestone))
		}
		uc.log("\n")
	}

	// 4. Usage Instructions
	uc.log("💡 HOW TO USE\n")
	if project.Analysis() != nil && len(project.Analysis().UsageInstructions()) > 0 {
		for _, instruction := range project.Analysis().UsageInstructions() {
			uc.log(fmt.Sprintf("   👉 %s\n", instruction))
		}
	} else {
		// Fallback instructions
		uc.log(fmt.Sprintf("   1. cd %s\n", localPath.String()))
		if project.Analysis() != nil && len(project.Analysis().EntryPoints()) > 0 {
			uc.log(fmt.Sprintf("   2. Run: %s\n", project.Analysis().EntryPoints()[0]))
		} else {
			uc.log("   2. Check README.md for run instructions\n")
		}
	}
	uc.log("\n")
	uc.log(strings.Repeat("=", 60) + "\n")

	return &SetupProjectOutput{
		Project:  project,
		Commands: commands,
		Warnings: warnings,
	}, nil
}

// Helper methods

func (uc *SetupProjectUseCase) log(message string) {
	fmt.Fprint(uc.output, message)
}

func (uc *SetupProjectUseCase) sendProgress(projectID, stage, message string, percentage int) {
	if uc.progressCB != nil {
		uc.progressCB(projectID, stage, message, percentage)
	}
}

func (uc *SetupProjectUseCase) readReadme(projectPath string) string {
	readmeFiles := []string{"README.md", "README.MD", "readme.md", "README.txt", "README", "Readme.md"}

	for _, readmeFile := range readmeFiles {
		path := filepath.Join(projectPath, readmeFile)
		if data, err := os.ReadFile(path); err == nil {
			return string(data)
		}
	}

	return ""
}

func (uc *SetupProjectUseCase) getFileList(projectPath string) []string {
	var files []string
	maxFiles := 100

	filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "__pycache__" ||
				name == "venv" || name == ".venv" || name == "target" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(projectPath, path)
		if err != nil {
			return nil
		}

		files = append(files, relPath)

		if len(files) >= maxFiles {
			return filepath.SkipAll
		}

		return nil
	})

	return files
}

// aiAnalysisResponse represents the AI's JSON response
type aiAnalysisResponse struct {
	ProjectType       string   `json:"project_type"`
	DetectedLanguages []string `json:"detected_languages"`
	PackageManagers   []string `json:"package_managers"`
	EntryPoints       []string `json:"entry_points"`
	Dependencies      []struct {
		Name      string `json:"name"`
		Version   string `json:"version"`
		Type      string `json:"type"`
		Installed bool   `json:"installed"`
	} `json:"dependencies"`
	SystemRequirements []string `json:"system_requirements"`
	Commands           []struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		Command     string `json:"command"`
		WorkingDir  string `json:"working_dir"`
		Stage       string `json:"stage"`
		Required    bool   `json:"required"`
		Status      string `json:"status"`
	} `json:"commands"`
	IsSuspicious      bool     `json:"is_suspicious"`
	SuspiciousReasons []string `json:"suspicious_reasons"`
	EstimatedSize     string   `json:"estimated_size"`
	Description       string   `json:"description"`
	UsageInstructions []string `json:"usage_instructions"`
	KeyMilestones     []string `json:"key_milestones"`
}

func (uc *SetupProjectUseCase) mergeAIAnalysis(analysis *analyzer.AnalysisResult, aiResponse string) error {
	// Try to extract JSON from response (AI might include markdown code blocks)
	jsonStr := aiResponse
	if start := strings.Index(aiResponse, "{"); start >= 0 {
		if end := strings.LastIndex(aiResponse, "}"); end > start {
			jsonStr = aiResponse[start : end+1]
		}
	}

	var aiResult aiAnalysisResponse
	if err := json.Unmarshal([]byte(jsonStr), &aiResult); err != nil {
		return fmt.Errorf("failed to parse AI response: %w", err)
	}

	// Merge AI results with basic analysis (AI takes precedence where available)
	if aiResult.ProjectType != "" {
		analysis.ProjectType = analyzer.ProjectType(aiResult.ProjectType)
	}
	if len(aiResult.DetectedLanguages) > 0 {
		analysis.DetectedLanguages = aiResult.DetectedLanguages
	}
	if len(aiResult.PackageManagers) > 0 {
		analysis.PackageManagers = aiResult.PackageManagers
	}
	if len(aiResult.EntryPoints) > 0 {
		analysis.EntryPoints = aiResult.EntryPoints
	}
	if len(aiResult.SystemRequirements) > 0 {
		analysis.SystemRequirements = aiResult.SystemRequirements
	}
	if aiResult.EstimatedSize != "" {
		analysis.EstimatedSize = aiResult.EstimatedSize
	}
	if aiResult.Description != "" {
		analysis.Description = aiResult.Description
	}
	if len(aiResult.UsageInstructions) > 0 {
		analysis.UsageInstructions = aiResult.UsageInstructions
	}
	if len(aiResult.KeyMilestones) > 0 {
		analysis.KeyMilestones = aiResult.KeyMilestones
	}

	// Merge dependencies
	if len(aiResult.Dependencies) > 0 {
		for _, dep := range aiResult.Dependencies {
			analysis.Dependencies = append(analysis.Dependencies, analyzer.Dependency{
				Name:      dep.Name,
				Version:   dep.Version,
				Type:      dep.Type,
				Installed: dep.Installed,
			})
		}
	}

	// Replace commands with AI-generated ones (they're smarter)
	if len(aiResult.Commands) > 0 {
		analysis.Commands = make([]analyzer.Command, 0, len(aiResult.Commands))
		for _, cmd := range aiResult.Commands {
			analysis.Commands = append(analysis.Commands, analyzer.Command{
				ID:          cmd.ID,
				Description: cmd.Description,
				Command:     cmd.Command,
				WorkingDir:  cmd.WorkingDir,
				Stage:       cmd.Stage,
				Required:    cmd.Required,
				Status:      cmd.Status,
			})
		}
	}

	return nil
}

func (uc *SetupProjectUseCase) convertAnalysisToDomain(analysis *analyzer.AnalysisResult, localPath *values.Path) *entities.Analysis {
	domainAnalysis := entities.NewAnalysis(entities.ProjectType(analysis.ProjectType))
	domainAnalysis.SetDetectedLanguages(analysis.DetectedLanguages)
	domainAnalysis.SetPackageManagers(analysis.PackageManagers)
	domainAnalysis.SetEntryPoints(analysis.EntryPoints)
	domainAnalysis.SetSystemRequirements(analysis.SystemRequirements)
	domainAnalysis.SetEstimatedSize(analysis.EstimatedSize)
	domainAnalysis.SetDescription(analysis.Description)
	domainAnalysis.SetUsageInstructions(analysis.UsageInstructions)
	domainAnalysis.SetKeyMilestones(analysis.KeyMilestones)

	// Convert dependencies
	dependencies := make([]entities.Dependency, 0, len(analysis.Dependencies))
	for _, dep := range analysis.Dependencies {
		dependencies = append(dependencies, entities.Dependency{
			Name:      dep.Name,
			Version:   dep.Version,
			Type:      dep.Type,
			Installed: dep.Installed,
		})
	}
	domainAnalysis.SetDependencies(dependencies)

	// Convert commands
	for _, cmd := range analysis.Commands {
		domainCmd, err := entities.NewCommand(
			cmd.ID,
			cmd.Description,
			cmd.Command,
			localPath,
			entities.CommandStage(cmd.Stage),
			cmd.Required,
		)
		if err == nil {
			domainAnalysis.AddCommand(domainCmd)
		}
	}

	return domainAnalysis
}

// resumeProject resumes project setup from where it left off
func (uc *SetupProjectUseCase) resumeProject(ctx context.Context, project *entities.Project, input SetupProjectInput) (*SetupProjectOutput, error) {
	projectID := project.ID().Value()
	uc.log(fmt.Sprintf("   📍 Current status: %s\n", project.Status()))

	// Determine what's been completed based on project state
	hasLocalPath := project.LocalPath() != nil
	hasAnalysis := project.Analysis() != nil
	currentStatus := project.Status()

	// Check if local path exists on disk
	localPathExists := false
	if hasLocalPath {
		localPathStr := project.LocalPath().String()
		if _, err := os.Stat(localPathStr); err == nil {
			localPathExists = true
		}
	}

	// Step 1: Parsing (always done if project exists)
	uc.log("📋 Step 1: Parsing GitHub URL...\n")
	uc.sendProgress(projectID, "parsing", "Using existing project", 100)
	uc.log("   ✓ Already parsed\n\n")

	// Step 2: AI availability check
	aiClient := uc.container.CreateAIClient()
	hasAI := aiClient != nil

	// Step 3: Clone - check if already cloned
	var localPath *values.Path
	if hasLocalPath && localPathExists {
		uc.log("📦 Step 3: Cloning repository...\n")
		uc.sendProgress(projectID, "cloning", "Already cloned", 100)
		uc.log(fmt.Sprintf("   ✓ Repository already cloned at: %s\n\n", project.LocalPath().String()))
		localPath = project.LocalPath()
	} else {
		// Need to clone
		uc.log("📦 Step 3: Cloning repository...\n")
		uc.sendProgress(projectID, "cloning", "Cloning repository", 20)

		// Check for cancellation
		select {
		case <-ctx.Done():
			if err := project.SetStatus(entities.StatusStopped); err == nil {
				uc.projectRepo.Save(ctx, project)
			}
			return nil, ctx.Err()
		default:
		}

		if err := project.SetStatus(entities.StatusCloning); err != nil {
			return nil, fmt.Errorf("failed to set status: %w", err)
		}
		uc.projectRepo.Save(ctx, project)

		gitClient, err := uc.container.CreateGitClient(project.URL().String())
		if err != nil {
			if err := project.SetStatus(entities.StatusFailed); err == nil {
				uc.projectRepo.Save(ctx, project)
			}
			return nil, fmt.Errorf("failed to create git client: %w", err)
		}

		if err := gitClient.Clone(input.WorkspaceDir, uc.output); err != nil {
			if err := project.SetStatus(entities.StatusFailed); err == nil {
				uc.projectRepo.Save(ctx, project)
			}
			return nil, fmt.Errorf("failed to clone repository: %w", err)
		}

		localPath, err = values.NewPath(gitClient.GetLocalPath())
		if err != nil {
			return nil, fmt.Errorf("invalid local path: %w", err)
		}

		if err := project.SetLocalPath(localPath); err != nil {
			return nil, fmt.Errorf("failed to set local path: %w", err)
		}

		uc.log(fmt.Sprintf("   ✓ Cloned to: %s\n\n", localPath.String()))
		uc.projectRepo.Save(ctx, project)
	}

	// Step 4: Security scan - always re-run (quick operation)
	select {
	case <-ctx.Done():
		if err := project.SetStatus(entities.StatusStopped); err == nil {
			uc.projectRepo.Save(ctx, project)
		}
		return nil, ctx.Err()
	default:
	}

	uc.log("🛡️  Step 4: Security scanning...\n")
	uc.sendProgress(projectID, "scanning", "Scanning for security issues", 40)

	scanner := uc.container.CreateScanner(localPath.String())
	scanResult, err := scanner.Scan()
	warnings := []string{}

	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Security scan failed: %v", err))
		uc.log(fmt.Sprintf("   ⚠️  Warning: Security scan failed: %v\n\n", err))
	} else {
		if scanResult.IsSuspicious {
			warnings = append(warnings, "Potentially suspicious code detected")
			uc.log("   ⚠️  WARNING: Potentially suspicious code detected!\n")
			for _, reason := range scanResult.SuspiciousReasons {
				uc.log(fmt.Sprintf("      - %s\n", reason))
				warnings = append(warnings, reason)
			}
			uc.log("\n")
		} else {
			uc.log("   ✓ No security issues detected\n")
		}
		uc.log(fmt.Sprintf("   ✓ Scanned %d files\n\n", len(scanResult.Findings)))
	}

	// Step 5: Analyze - check if already analyzed
	var analysisResult *analyzer.AnalysisResult
	if hasAnalysis && currentStatus != entities.StatusPending && currentStatus != entities.StatusCloning {
		uc.log("🔍 Step 5: Analyzing project...\n")
		uc.sendProgress(projectID, "analyzing", "Using existing analysis", 100)
		uc.log("   ✓ Using existing analysis\n\n")

		// Convert existing analysis back to analyzer result
		existingAnalysis := project.Analysis()
		analysisResult = &analyzer.AnalysisResult{
			ProjectType:        analyzer.ProjectType(existingAnalysis.ProjectType()),
			DetectedLanguages:  existingAnalysis.DetectedLanguages(),
			PackageManagers:    existingAnalysis.PackageManagers(),
			EntryPoints:        existingAnalysis.EntryPoints(),
			SystemRequirements: existingAnalysis.SystemRequirements(),
			EstimatedSize:      existingAnalysis.EstimatedSize(),
			Description:        existingAnalysis.Description(),
		}
	} else {
		// Need to analyze
		select {
		case <-ctx.Done():
			if err := project.SetStatus(entities.StatusStopped); err == nil {
				uc.projectRepo.Save(ctx, project)
			}
			return nil, ctx.Err()
		default:
		}

		uc.log("🔍 Step 5: Analyzing project...\n")
		uc.sendProgress(projectID, "analyzing", "Analyzing project structure", 50)

		if err := project.SetStatus(entities.StatusAnalyzing); err != nil {
			return nil, fmt.Errorf("failed to set status: %w", err)
		}
		uc.projectRepo.Save(ctx, project)

		projectAnalyzer := uc.container.CreateAnalyzer(localPath.String())
		var err error
		analysisResult, err = projectAnalyzer.Analyze()
		if err != nil {
			if err := project.SetStatus(entities.StatusFailed); err == nil {
				uc.projectRepo.Save(ctx, project)
			}
			return nil, fmt.Errorf("failed to analyze project: %w", err)
		}

		project.SetType(entities.ProjectType(analysisResult.ProjectType))
		uc.log(fmt.Sprintf("   ✓ Project type: %s\n", analysisResult.ProjectType))
		uc.log(fmt.Sprintf("   ✓ Package managers: %v\n", analysisResult.PackageManagers))
		uc.log(fmt.Sprintf("   ✓ Entry points: %v\n", analysisResult.EntryPoints))
		uc.log(fmt.Sprintf("   ✓ Dependencies: %d\n\n", len(analysisResult.Dependencies)))

		// Run AI analysis if available
		if hasAI {
			uc.log("🤖 Running AI analysis...\n")
			uc.sendProgress(projectID, "ai_analysis", "AI analyzing", 60)

			readmeContent := uc.readReadme(localPath.String())
			fileList := uc.getFileList(localPath.String())

			aiResponse, err := aiClient.AnalyzeProject(ctx, localPath.String(), readmeContent, fileList)
			if err != nil {
				uc.log(fmt.Sprintf("   ⚠️  AI analysis failed: %v\n", err))
				uc.log("   ℹ️  Falling back to basic analysis - the project will still work!\n\n")
			} else {
				uc.log("   ✓ AI analysis completed\n")
				if err := uc.mergeAIAnalysis(analysisResult, aiResponse); err != nil {
					uc.log(fmt.Sprintf("   ⚠️  Failed to merge AI analysis: %v\n", err))
					uc.log("   ℹ️  Using basic analysis instead\n")
				} else {
					uc.log("   ✓ Enhanced with AI insights\n")
					if analysisResult.Description != "" {
						uc.log(fmt.Sprintf("   ✓ Description: %s\n", analysisResult.Description))
					}
				}
				uc.log("\n")
			}
			uc.sendProgress(projectID, "ai_analysis", "AI analysis complete", 100)
		}

		// Convert analysis result to domain entity and save
		domainAnalysis := uc.convertAnalysisToDomain(analysisResult, localPath)
		project.SetAnalysis(domainAnalysis)
		uc.projectRepo.Save(ctx, project)
	}

	// Step 6: Execute setup commands
	select {
	case <-ctx.Done():
		if err := project.SetStatus(entities.StatusStopped); err == nil {
			uc.projectRepo.Save(ctx, project)
		}
		return nil, ctx.Err()
	default:
	}

	uc.log("⚙️  Step 6: Setting up project...\n")
	uc.sendProgress(projectID, "installing", "Installing dependencies", 60)

	if err := project.SetStatus(entities.StatusInstalling); err != nil {
		return nil, fmt.Errorf("failed to set status: %w", err)
	}
	uc.projectRepo.Save(ctx, project)

	// Convert analysis result to domain entity if not already done
	var domainAnalysis *entities.Analysis
	if project.Analysis() != nil {
		domainAnalysis = project.Analysis()
	} else {
		domainAnalysis = uc.convertAnalysisToDomain(analysisResult, localPath)
		project.SetAnalysis(domainAnalysis)
		uc.projectRepo.Save(ctx, project)
	}

	commands := domainAnalysis.Commands()
	if len(commands) > 0 {
		// Create output handler that clears spinner before writing
		firstOutput := true
		outputHandler := func(line string) {
			// Clear spinner line before first output
			if firstOutput {
				fmt.Fprint(uc.output, "\r\033[2K")
				firstOutput = false
			}
			// Ensure proper line formatting
			if !strings.HasSuffix(line, "\n") {
				fmt.Fprint(uc.output, line+"\n")
			} else {
				fmt.Fprint(uc.output, line)
			}
		}

		executeUC := execution.NewExecuteCommandsUseCase(uc.container, outputHandler)

		// Set AI client for auto-fix functionality (use container to create one)
		resumeAIClient := uc.container.CreateAIClient()
		if resumeAIClient != nil {
			executeUC.SetAIClient(resumeAIClient)
		}

		if uc.progressCB != nil {
			executeUC.SetProgressCallback(execution.ProgressCallback(uc.progressCB))
		}

		execCommands := make([]*executor.Command, 0, len(commands))
		for _, cmd := range commands {
			execCommands = append(execCommands, &executor.Command{
				ID:          cmd.ID(),
				Description: cmd.Description(),
				Command:     cmd.Command(),
				WorkingDir:  cmd.WorkingDir().String(),
				Stage:       string(cmd.Stage()),
				Required:    cmd.Required(),
				Status:      executor.CommandStatus(cmd.Status()),
			})
		}

		err := executeUC.Execute(ctx, execution.ExecuteCommandsInput{
			Commands:   execCommands,
			WorkingDir: localPath.String(),
			Mode:       input.Mode,
		})

		if err != nil {
			if err := project.SetStatus(entities.StatusFailed); err == nil {
				uc.projectRepo.Save(ctx, project)
			}
			return nil, fmt.Errorf("failed to execute commands: %w", err)
		}

		// Update command statuses from executor
		for i, execCmd := range execCommands {
			if i < len(commands) {
				cmd := commands[i]
				switch execCmd.Status {
				case executor.CommandCompleted:
					cmd.MarkCompleted(execCmd.Output)
				case executor.CommandFailed:
					cmd.MarkFailed(execCmd.Output, execCmd.Error, execCmd.ExitCode)
				case executor.CommandSkipped:
					cmd.MarkSkipped()
				}
			}
		}
	} else {
		uc.log("   ℹ️  No setup commands detected\n\n")
	}

	// Step 7: Complete
	uc.log("✅ Step 7: Project ready!\n")
	uc.sendProgress(projectID, "ready", "Project is ready", 100)

	if err := project.SetStatus(entities.StatusReady); err != nil {
		return nil, fmt.Errorf("failed to set status: %w", err)
	}
	uc.projectRepo.Save(ctx, project)

	uc.log("=" + strings.Repeat("=", 50) + "\n")
	uc.log(fmt.Sprintf("🎉 Project '%s' is ready to use!\n", project.Name()))
	uc.log(fmt.Sprintf("📁 Location: %s\n\n", localPath.String()))

	return &SetupProjectOutput{
		Project:  project,
		Commands: commands,
		Warnings: warnings,
	}, nil
}
