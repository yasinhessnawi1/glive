package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/glive/core/config"
	"github.com/glive/infrastructure/container"
	"github.com/glive/infrastructure/executor"
	"github.com/glive/interface/cli"
	"github.com/glive/interface/controller"
	"github.com/glive/interface/tui"
	_ "github.com/glive/interface/tui/views" // Import for init() side effects
	"github.com/glive/usecase/project"
	"github.com/spf13/cobra"
)

var (
	cfgFile      string
	mode         string
	verbose      bool
	force        bool
	noTUI        bool
	aiMode       string
	aiConfidence float64
	noFallback   bool
	noAIRecovery bool
	// AI Recovery flags
	maxRecoveryAttempts int
	autoApproveRisk     string
	// Sandbox flags
	sandboxEnabled bool
	sandboxRoot    string
	noNetwork      bool
	cpuLimit       string
	memoryLimit    string
	sandboxTimeout string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "glive <github-url>",
	Short: "GLive - GitHub to Live",
	Long: `GLive automatically clones and sets up any GitHub project.

It analyzes the project, detects dependencies, and generates
the correct setup commands for your platform.

EXECUTION MODES:
  auto      Automatically execute all setup commands (default)
  assisted  Prompt for approval before each command
  manual    Only analyze and show commands, don't execute

AI-FIRST MODES:
  ai-first     Use AI as primary decision maker (default)
  traditional  Use pattern-based detection
  hybrid       AI analysis + traditional execution

EXAMPLES:
  # Set up a project automatically (AI-first mode)
  glive run https://github.com/user/repo

  # Set up with traditional mode
  glive run https://github.com/user/repo --ai-mode traditional

  # Set up with custom AI confidence threshold
  glive run https://github.com/user/repo --ai-confidence 0.8

  # Set up with manual approval for each step
  glive run https://github.com/user/repo --mode assisted

  # Just analyze without running commands
  glive run https://github.com/user/repo --mode manual
  
  # Disable fallback (fail if AI fails)
  glive run https://github.com/user/repo --no-fallback
  
  # Disable AI recovery
  glive run https://github.com/user/repo --no-ai-recovery

  # Use shorthand repository format
  glive user/repo

  # Force re-clone even if project exists
  glive user/repo --force

  # Configure your AI API key
  glive config set api-key sk-xxx

  # View project status
  glive status

  # List all projects
  glive list

  # Clean up old projects
  glive cleanup`,
	Args:                       cobra.MaximumNArgs(1),
	DisableFlagParsing:         false,
	DisableSuggestions:         false,
	SuggestionsMinimumDistance: 2,
	Run:                        runProject,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.glive.json)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&noTUI, "no-tui", false, "disable TUI and use basic CLI")

	// Command-specific flags
	rootCmd.Flags().StringVarP(&mode, "mode", "m", "auto", "execution mode: auto, assisted, manual")
	rootCmd.Flags().BoolVarP(&force, "force", "f", false, "force re-clone even if project exists")

	// AI-first flags
	rootCmd.Flags().StringVar(&aiMode, "ai-mode", "ai-first", "AI execution mode: ai-first, traditional, hybrid")
	rootCmd.Flags().Float64Var(&aiConfidence, "ai-confidence", 0.7, "AI confidence threshold (0.0-1.0)")
	rootCmd.Flags().BoolVar(&noFallback, "no-fallback", false, "disable fallback to traditional mode")
	rootCmd.Flags().BoolVar(&noAIRecovery, "no-ai-recovery", false, "disable AI-powered recovery")

	// AI Recovery configuration flags
	rootCmd.Flags().IntVar(&maxRecoveryAttempts, "max-recovery-attempts", 3, "maximum number of recovery attempts per command")
	rootCmd.Flags().StringVar(&autoApproveRisk, "auto-approve-risk", "low", "auto-approve recovery steps up to this risk level (low/medium/high/critical)")

	// Update help text to include recovery examples
	rootCmd.Long = rootCmd.Long + `

AI RECOVERY OPTIONS:
  --max-recovery-attempts N    Maximum recovery attempts (default: 3)
  --auto-approve-risk LEVEL    Auto-approve up to risk level (default: low)
  
EXAMPLES:
  # Set up with custom recovery settings
  glive user/repo --max-recovery-attempts 5 --auto-approve-risk medium`

	// Sandbox flags
	rootCmd.Flags().BoolVar(&sandboxEnabled, "sandbox", false, "enable sandbox mode for command execution")
	rootCmd.Flags().StringVar(&sandboxRoot, "sandbox-root", "", "custom sandbox root directory (default: $TMPDIR/glive-sandbox)")
	rootCmd.Flags().BoolVar(&noNetwork, "no-network", false, "disable network access in sandbox")
	rootCmd.Flags().StringVar(&cpuLimit, "cpu-limit", "", "CPU limit (e.g., \"1\", \"50%\")")
	rootCmd.Flags().StringVar(&memoryLimit, "memory-limit", "", "Memory limit (e.g., \"2GB\", \"512MB\")")
	rootCmd.Flags().StringVar(&sandboxTimeout, "sandbox-timeout", "", "Maximum execution time for sandboxed commands (e.g., \"30m\", \"1h\")")

	// Add subcommands
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(cleanupCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(versionCmd)
}

func runProject(cmd *cobra.Command, args []string) {
	// If no args and TUI enabled, launch TUI dashboard
	if len(args) == 0 && !noTUI {
		runTUI()
		return
	}

	// If no args, show help
	if len(args) == 0 {
		_ = cmd.Help()
		return
	}

	githubURL := args[0]

	// If TUI enabled and we have a URL, still use TUI for better UX
	if !noTUI {
		runTUIWithProject(githubURL)
		return
	}

	// Fallback to basic CLI
	runBasicCLI(githubURL)
}

// buildFlagsMap converts CLI flags to a configuration map
func buildFlagsMap() map[string]interface{} {
	flags := make(map[string]interface{})

	// Always include mode flag if set (flags override config)
	// Note: Cobra sets default to "auto", so we include it to ensure
	// flag values override config values even when they match defaults
	if mode != "" {
		flags["default-mode"] = mode
	}

	// AI-first flags
	if aiMode != "" {
		flags["ai-first-mode"] = aiMode
	}
	if aiConfidence > 0 {
		flags["ai-confidence"] = aiConfidence
	}
	if noFallback {
		flags["enable-fallback"] = false
	}
	if noAIRecovery {
		flags["enable-auto-recovery"] = false
	}

	// AI Recovery configuration flags
	if maxRecoveryAttempts > 0 {
		flags["max-recovery-attempts"] = maxRecoveryAttempts
	}
	if autoApproveRisk != "" {
		flags["auto-approve-risk"] = autoApproveRisk
	}

	// Sandbox flags
	if sandboxEnabled {
		flags["enable-sandbox"] = true
	}
	if sandboxRoot != "" {
		flags["sandbox-root"] = sandboxRoot
	}
	if noNetwork {
		flags["sandbox-no-network"] = true
	}
	if cpuLimit != "" {
		flags["sandbox-cpu-limit"] = cpuLimit
	}
	if memoryLimit != "" {
		flags["sandbox-memory-limit"] = memoryLimit
	}
	if sandboxTimeout != "" {
		flags["sandbox-timeout"] = sandboxTimeout
	}

	return flags
}

// runTUI launches the TUI dashboard
func runTUI() {
	// Build flags map from CLI flags
	flags := buildFlagsMap()

	// Load configuration with flags
	cfgManager, err := config.NewWithFlags(flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", cli.Error("Failed to load configuration: "+err.Error()))
		os.Exit(1)
	}

	cfg := cfgManager.Get()

	// Initialize workspace
	if err := cfgManager.InitWorkspace(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", cli.Error("Failed to initialize workspace: "+err.Error()))
		os.Exit(1)
	}

	// Parse execution mode (use from config or flag)
	executionMode := executor.ExecutionMode(cfg.DefaultMode)
	if mode != "" && mode != "auto" {
		executionMode = executor.ExecutionMode(mode)
	}
	if executionMode != executor.ModeAuto && executionMode != executor.ModeAssisted && executionMode != executor.ModeManual {
		executionMode = executor.ModeAuto
	}

	// Create container config
	containerCfg := &container.Config{
		APIKey:        cfg.APIKey,
		APIProvider:   cfg.APIProvider,
		APIEndpoint:   cfg.APIEndpoint,
		WorkspaceDir:  cfg.WorkspaceDir,
		DefaultMode:   executionMode,
		MaxConcurrent: cfg.MaxConcurrent,
		EnableSandbox: cfg.EnableSandbox,
	}

	// Create logger
	logger := container.NewSimpleLogger(os.Stdout)

	// Create dependency injection container
	cont, err := container.NewContainer(containerCfg, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", cli.Error("Failed to create container: "+err.Error()))
		os.Exit(1)
	}

	// Create and run TUI
	app, err := tui.NewApp(cont)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", cli.Error("Failed to create TUI: "+err.Error()))
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", cli.Error("TUI error: "+err.Error()))
		os.Exit(1)
	}
}

// runTUIWithProject launches TUI with a project to setup
func runTUIWithProject(githubURL string) {
	// Build flags map from CLI flags
	flags := buildFlagsMap()

	// Load configuration with flags
	cfgManager, err := config.NewWithFlags(flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", cli.Error("Failed to load configuration: "+err.Error()))
		os.Exit(1)
	}

	cfg := cfgManager.Get()

	// Initialize workspace
	if err := cfgManager.InitWorkspace(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", cli.Error("Failed to initialize workspace: "+err.Error()))
		os.Exit(1)
	}

	// Create container config
	containerCfg := &container.Config{
		APIKey:        cfg.APIKey,
		APIProvider:   cfg.APIProvider,
		APIEndpoint:   cfg.APIEndpoint,
		WorkspaceDir:  cfg.WorkspaceDir,
		DefaultMode:   executor.ExecutionMode(cfg.DefaultMode),
		MaxConcurrent: cfg.MaxConcurrent,
		EnableSandbox: cfg.EnableSandbox,
	}

	// Create logger
	logger := container.NewSimpleLogger(os.Stdout)

	// Create dependency injection container
	cont, err := container.NewContainer(containerCfg, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", cli.Error("Failed to create container: "+err.Error()))
		os.Exit(1)
	}

	// Create and run TUI  - for now start with execution view showing the github URL
	app, err := tui.NewApp(cont)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", cli.Error("Failed to create TUI: "+err.Error()))
		os.Exit(1)
	}

	// Start TUI in execution view for the project
	// The execution will be shown in the TUI
	if err := app.RunWithProjectOptions(githubURL, force); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", cli.Error("TUI error: "+err.Error()))
		os.Exit(1)
	}
}

// runBasicCLI runs the basic CLI (original implementation)
func runBasicCLI(githubURL string) {
	// Build flags map from CLI flags
	flags := buildFlagsMap()

	// Load configuration with flags
	cfgManager, err := config.NewWithFlags(flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", cli.Error("Failed to load configuration: "+err.Error()))
		os.Exit(1)
	}

	cfg := cfgManager.Get()

	// Initialize workspace
	if err := cfgManager.InitWorkspace(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", cli.Error("Failed to initialize workspace: "+err.Error()))
		os.Exit(1)
	}

	// Parse execution mode (use from config or flag)
	executionMode := executor.ExecutionMode(cfg.DefaultMode)
	if mode != "" && mode != "auto" {
		executionMode = executor.ExecutionMode(mode)
	}
	if executionMode != executor.ModeAuto && executionMode != executor.ModeAssisted && executionMode != executor.ModeManual {
		fmt.Fprintf(os.Stderr, "%s\n", cli.Error(fmt.Sprintf("Invalid mode: %s (must be: auto, assisted, or manual)", mode)))
		os.Exit(1)
	}

	// Create container config from core config
	containerCfg := &container.Config{
		APIKey:        cfg.APIKey,
		APIProvider:   cfg.APIProvider,
		APIEndpoint:   cfg.APIEndpoint,
		WorkspaceDir:  cfg.WorkspaceDir,
		DefaultMode:   executionMode,
		MaxConcurrent: cfg.MaxConcurrent,
		EnableSandbox: cfg.EnableSandbox,
	}

	// Create logger
	logger := container.NewSimpleLogger(os.Stdout)

	// Create dependency injection container
	cont, err := container.NewContainer(containerCfg, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", cli.Error("Failed to create container: "+err.Error()))
		os.Exit(1)
	}

	// Create terminal UI manager
	ui := cli.NewTerminalUI(os.Stdout, 7) // 7 steps total
	defer ui.Close()

	// Create UI-aware writer that routes output through terminal UI
	uiWriter := cli.NewUIWriter(ui)

	// Create CLI controller (Clean Architecture)
	cliController := controller.NewCLIController(cont, uiWriter)

	// Set up progress callback with proper terminal UI and debouncing
	currentStage := ""
	lastPercentage := -1
	lastUpdateTime := time.Time{}
	progressCallback := func(projectID string, stage string, message string, percentage int) {
		// Map stages to steps
		stageToStep := map[string]struct {
			num   int
			emoji string
			title string
		}{
			"parsing":     {1, "📋", "Parsing GitHub URL"},
			"cloning":     {2, "📦", "Cloning repository"},
			"scanning":    {3, "🛡️", "Security scanning"},
			"analyzing":   {4, "🔍", "Analyzing project"},
			"ai_analysis": {5, "🤖", "AI-powered analysis"},
			"installing":  {6, "⚙️", "Setting up project"},
			"executing":   {6, "⚙️", "Setting up project"}, // executing is part of installing
			"ready":       {7, "✅", "Project ready"},
		}

		stepInfo, exists := stageToStep[stage]
		if !exists {
			return
		}

		// Start new step if stage changed
		if stage != currentStage {
			if currentStage != "" {
				ui.CompleteStep() // Complete previous step
			}
			currentStage = stage
			lastPercentage = -1          // Reset percentage tracking for new stage
			lastUpdateTime = time.Time{} // Reset time tracking
			ui.StartStep(stepInfo.num, stepInfo.emoji, stepInfo.title, message)
		}

		// Show progress bar for installation/executing stages
		if stage == "installing" || stage == "executing" {
			now := time.Now()
			// Aggressive debouncing: only update if:
			// 1. Percentage changed by at least 5% OR
			// 2. At least 500ms passed since last update OR
			// 3. Progress is complete (100%)
			percentDiff := 0
			if lastPercentage >= 0 {
				if percentage > lastPercentage {
					percentDiff = percentage - lastPercentage
				} else {
					percentDiff = lastPercentage - percentage
				}
			}
			timeSinceLastUpdate := now.Sub(lastUpdateTime)

			shouldUpdateProgress := lastPercentage < 0 || // First update
				percentDiff >= 5 || // Significant change
				timeSinceLastUpdate >= 500*time.Millisecond || // Time threshold
				percentage >= 100 // Complete

			if percentage >= 100 {
				// Installation complete
				ui.CompleteStep("Installation complete")
				lastPercentage = 100
				lastUpdateTime = now
			} else if shouldUpdateProgress {
				// Update progress bar (internal throttling will handle further smoothing)
				ui.ShowProgress(percentage, 100, message)
				lastPercentage = percentage
				lastUpdateTime = now
			}
		} else {
			// For other stages, just update message occasionally
			now := time.Now()
			if lastUpdateTime.IsZero() || now.Sub(lastUpdateTime) >= 300*time.Millisecond {
				ui.UpdateStepMessage(message)
				lastUpdateTime = now
			}
		}
	}

	cliController.SetProgressCallback(project.ProgressCallback(progressCallback))

	// Run the project using Clean Architecture
	ctx := context.Background()

	// Determine AI mode from config or flags
	aiModeValue := "ai-first" // Default
	if cfg.AIFirst != nil && cfg.AIFirst.PrimaryMode != "" {
		aiModeValue = cfg.AIFirst.PrimaryMode
	}
	if aiMode != "" {
		aiModeValue = aiMode
	}

	aiConfidenceValue := 0.7 // Default
	if cfg.AIFirst != nil && cfg.AIFirst.ConfidenceThreshold > 0 {
		aiConfidenceValue = cfg.AIFirst.ConfidenceThreshold
	}
	if aiConfidence > 0 {
		aiConfidenceValue = aiConfidence
	}

	noFallbackValue := false
	if cfg.AIFirst != nil {
		noFallbackValue = !cfg.AIFirst.EnableFallback
	}
	if noFallback {
		noFallbackValue = true
	}

	noAIRecoveryValue := false
	if cfg.AIFirst != nil {
		noAIRecoveryValue = !cfg.AIFirst.EnableAutoRecovery
	}
	if noAIRecovery {
		noAIRecoveryValue = true
	}

	err = cliController.RunProjectWithAIOptions(ctx, githubURL, executionMode, cfg.WorkspaceDir, force, aiModeValue, aiConfidenceValue, noFallbackValue, noAIRecoveryValue)
	if err != nil {
		ui.FailStep(err.Error())
		os.Exit(1)
	}

	// Complete final step
	ui.CompleteStep("Setup complete!")
}
