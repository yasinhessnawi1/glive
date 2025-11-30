package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [project-path]",
	Short: "Run a project",
	Long: `Run a project that has been set up with glive.

If no project path is provided, runs the project in the current directory.

Examples:
  glive run                     # Run project in current directory
  glive run ./my-project        # Run project in specified directory
  glive run ~/projects/myapp    # Run project with absolute path`,
	Args: cobra.MaximumNArgs(1),
	Run:  executeRun,
}

var (
	runDetach bool
)

func init() {
	runCmd.Flags().BoolVarP(&runDetach, "detach", "d", false, "Run in background (detached mode)")
}

func executeRun(cmd *cobra.Command, args []string) {
	// Determine project path
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		fmt.Printf("Error resolving path: %v\n", err)
		os.Exit(1)
	}

	// Check if directory exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		fmt.Printf("Project directory not found: %s\n", absPath)
		os.Exit(1)
	}

	// Detect project type and get start command
	startCmd, projectType := detectStartCommand(absPath)
	if startCmd == "" {
		fmt.Printf("Could not detect how to run this project.\n")
		fmt.Printf("Project type: %s\n", projectType)
		fmt.Println("\nTry one of these commands manually:")
		printManualCommands(projectType)
		os.Exit(1)
	}

	fmt.Printf("Starting %s project...\n", projectType)
	fmt.Printf("Command: %s\n", startCmd)
	fmt.Printf("Directory: %s\n\n", absPath)

	// Parse the command
	parts := strings.Fields(startCmd)
	if len(parts) == 0 {
		fmt.Println("Invalid start command")
		os.Exit(1)
	}

	// Create the command
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	execCmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	execCmd.Dir = absPath
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Stdin = os.Stdin

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nStopping project...")
		cancel()
	}()

	// Run the command
	if err := execCmd.Run(); err != nil {
		if ctx.Err() == context.Canceled {
			fmt.Println("Project stopped.")
		} else {
			fmt.Printf("Error running project: %v\n", err)
			os.Exit(1)
		}
	}
}

func detectStartCommand(projectPath string) (string, string) {
	// Check for package.json (Node.js)
	if _, err := os.Stat(filepath.Join(projectPath, "package.json")); err == nil {
		startCmd := detectNodeStartCommand(projectPath)
		return startCmd, "Node.js"
	}

	// Check for go.mod (Go)
	if _, err := os.Stat(filepath.Join(projectPath, "go.mod")); err == nil {
		startCmd := detectGoStartCommand(projectPath)
		return startCmd, "Go"
	}

	// Check for requirements.txt or setup.py (Python)
	if _, err := os.Stat(filepath.Join(projectPath, "requirements.txt")); err == nil {
		startCmd := detectPythonStartCommand(projectPath)
		return startCmd, "Python"
	}
	if _, err := os.Stat(filepath.Join(projectPath, "setup.py")); err == nil {
		startCmd := detectPythonStartCommand(projectPath)
		return startCmd, "Python"
	}
	if _, err := os.Stat(filepath.Join(projectPath, "pyproject.toml")); err == nil {
		startCmd := detectPythonStartCommand(projectPath)
		return startCmd, "Python"
	}

	// Check for Cargo.toml (Rust)
	if _, err := os.Stat(filepath.Join(projectPath, "Cargo.toml")); err == nil {
		return "cargo run", "Rust"
	}

	// Check for pom.xml (Java/Maven)
	if _, err := os.Stat(filepath.Join(projectPath, "pom.xml")); err == nil {
		return "mvn spring-boot:run", "Java (Maven)"
	}

	// Check for build.gradle (Java/Gradle)
	if _, err := os.Stat(filepath.Join(projectPath, "build.gradle")); err == nil {
		return "gradle run", "Java (Gradle)"
	}

	// Check for docker-compose.yml
	if _, err := os.Stat(filepath.Join(projectPath, "docker-compose.yml")); err == nil {
		return "docker-compose up", "Docker"
	}
	if _, err := os.Stat(filepath.Join(projectPath, "docker-compose.yaml")); err == nil {
		return "docker-compose up", "Docker"
	}

	// Check for Dockerfile
	if _, err := os.Stat(filepath.Join(projectPath, "Dockerfile")); err == nil {
		return "docker build -t app . && docker run -it app", "Docker"
	}

	return "", "Unknown"
}

func detectNodeStartCommand(projectPath string) string {
	// Read package.json to find start script
	content, err := os.ReadFile(filepath.Join(projectPath, "package.json"))
	if err != nil {
		return "npm start"
	}

	contentStr := string(content)

	// Check for various start scripts
	if strings.Contains(contentStr, `"dev"`) {
		// Check which package manager
		if _, err := os.Stat(filepath.Join(projectPath, "pnpm-lock.yaml")); err == nil {
			return "pnpm dev"
		}
		if _, err := os.Stat(filepath.Join(projectPath, "yarn.lock")); err == nil {
			return "yarn dev"
		}
		return "npm run dev"
	}

	if strings.Contains(contentStr, `"start"`) {
		if _, err := os.Stat(filepath.Join(projectPath, "pnpm-lock.yaml")); err == nil {
			return "pnpm start"
		}
		if _, err := os.Stat(filepath.Join(projectPath, "yarn.lock")); err == nil {
			return "yarn start"
		}
		return "npm start"
	}

	return "npm start"
}

func detectGoStartCommand(projectPath string) string {
	// Check for main.go in root
	if _, err := os.Stat(filepath.Join(projectPath, "main.go")); err == nil {
		return "go run main.go"
	}

	// Check for cmd directory
	cmdDir := filepath.Join(projectPath, "cmd")
	if entries, err := os.ReadDir(cmdDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				return fmt.Sprintf("go run ./cmd/%s", entry.Name())
			}
		}
	}

	return "go run ."
}

func detectPythonStartCommand(projectPath string) string {
	// Common Python entry points
	entryPoints := []string{"main.py", "app.py", "run.py", "server.py", "manage.py"}

	for _, ep := range entryPoints {
		if _, err := os.Stat(filepath.Join(projectPath, ep)); err == nil {
			if ep == "manage.py" {
				return "python manage.py runserver"
			}
			return fmt.Sprintf("python %s", ep)
		}
	}

	// Check for Flask or FastAPI in requirements.txt
	content, err := os.ReadFile(filepath.Join(projectPath, "requirements.txt"))
	if err == nil {
		contentStr := strings.ToLower(string(content))
		if strings.Contains(contentStr, "flask") {
			return "flask run"
		}
		if strings.Contains(contentStr, "fastapi") || strings.Contains(contentStr, "uvicorn") {
			return "uvicorn main:app --reload"
		}
	}

	return "python main.py"
}

func printManualCommands(projectType string) {
	switch projectType {
	case "Node.js":
		fmt.Println("  npm start")
		fmt.Println("  npm run dev")
		fmt.Println("  yarn start")
		fmt.Println("  pnpm dev")
	case "Go":
		fmt.Println("  go run main.go")
		fmt.Println("  go run .")
		fmt.Println("  go run ./cmd/<app>")
	case "Python":
		fmt.Println("  python main.py")
		fmt.Println("  python app.py")
		fmt.Println("  flask run")
		fmt.Println("  uvicorn main:app --reload")
	case "Rust":
		fmt.Println("  cargo run")
	case "Java (Maven)":
		fmt.Println("  mvn spring-boot:run")
		fmt.Println("  mvn exec:java")
	case "Java (Gradle)":
		fmt.Println("  gradle run")
		fmt.Println("  ./gradlew run")
	case "Docker":
		fmt.Println("  docker-compose up")
		fmt.Println("  docker build -t app . && docker run -it app")
	default:
		fmt.Println("  Check the project's README.md for instructions")
	}
}
