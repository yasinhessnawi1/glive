package analyzer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/glive/core/types"
)

// Analyzer analyzes project structure and determines setup requirements
type Analyzer struct {
	projectPath string
}

// New creates a new Analyzer instance
// IMPORTANT: projectPath must be a valid, non-empty absolute path
func New(projectPath string) *Analyzer {
	// Safety: If projectPath is empty, this would analyze current directory
	// which could be the GLive agent itself - this is a bug that should be caught earlier
	if projectPath == "" {
		// Return analyzer that will fail on Analyze()
		return &Analyzer{
			projectPath: "",
		}
	}
	return &Analyzer{
		projectPath: projectPath,
	}
}

// Analyze performs project analysis
func (a *Analyzer) Analyze() (*types.AnalysisResult, error) {
	// Safety check: Ensure projectPath is set
	if a.projectPath == "" {
		return nil, fmt.Errorf("project path is empty: cannot analyze without a valid path")
	}

	result := &types.AnalysisResult{
		DetectedLanguages:  []string{},
		PackageManagers:    []string{},
		EntryPoints:        []string{},
		Dependencies:       []types.Dependency{},
		SystemRequirements: []string{},
		Commands:           []types.Command{},
		IsSuspicious:       false,
		SuspiciousReasons:  []string{},
	}

	// Detect project type
	projectType, detectedTypes := a.detectProjectType()
	result.ProjectType = projectType

	// Detect package managers and dependencies
	if err := a.detectPackageManagers(result); err != nil {
		return nil, fmt.Errorf("failed to detect package managers: %w", err)
	}

	// Find entry points
	if err := a.detectEntryPoints(result); err != nil {
		return nil, fmt.Errorf("failed to detect entry points: %w", err)
	}

	// Analyze README
	if err := a.analyzeReadme(result); err != nil {
		// README analysis is optional, log but don't fail
		fmt.Printf("Warning: README analysis failed: %v\n", err)
	}

	// Estimate project size
	result.EstimatedSize = a.estimateSize()

	// Generate setup commands based on detected type
	a.generateCommands(result, detectedTypes)

	return result, nil
}

// detectProjectType determines the primary project type
func (a *Analyzer) detectProjectType() (types.ProjectType, []types.ProjectType) {
	detectedTypes := []types.ProjectType{}

	// Check for Node.js
	if a.fileExists("package.json") {
		detectedTypes = append(detectedTypes, types.ProjectTypeNodeJS)
	}

	// Check for Python
	if a.fileExists("requirements.txt") || a.fileExists("setup.py") ||
		a.fileExists("pyproject.toml") || a.fileExists("Pipfile") {
		detectedTypes = append(detectedTypes, types.ProjectTypePython)
	}

	// Check for Go
	if a.fileExists("go.mod") {
		detectedTypes = append(detectedTypes, types.ProjectTypeGo)
	}

	// Check for Rust
	if a.fileExists("Cargo.toml") {
		detectedTypes = append(detectedTypes, types.ProjectTypeRust)
	}

	// Check for Java
	if a.fileExists("pom.xml") || a.fileExists("build.gradle") {
		detectedTypes = append(detectedTypes, types.ProjectTypeJava)
	}

	// Check for Docker
	if a.fileExists("Dockerfile") || a.fileExists("docker-compose.yml") {
		detectedTypes = append(detectedTypes, types.ProjectTypeDocker)
	}

	// Determine primary type
	if len(detectedTypes) == 0 {
		return types.ProjectTypeUnknown, detectedTypes
	} else if len(detectedTypes) == 1 {
		return detectedTypes[0], detectedTypes
	} else {
		// Polyglot project - prioritize based on common patterns
		return types.ProjectTypePolyglot, detectedTypes
	}
}

// detectPackageManagers identifies package managers in use
func (a *Analyzer) detectPackageManagers(result *types.AnalysisResult) error {
	// Node.js package managers - check lock files first (most reliable indicator)
	if a.fileExists("package.json") {
		// Check package.json for packageManager field (Node.js 16.9+) - most authoritative
		hasPackageManagerField := false
		data, err := os.ReadFile(filepath.Join(a.projectPath, "package.json"))
		if err == nil {
			var pkg map[string]interface{}
			if json.Unmarshal(data, &pkg) == nil {
				if pkgManager, ok := pkg["packageManager"].(string); ok {
					hasPackageManagerField = true
					// Extract package manager from format "npm@8.1.0", "yarn@3.2.0", "pnpm@7.0.0", "bun@1.0.0"
					pkgManagerLower := strings.ToLower(pkgManager)
					if strings.HasPrefix(pkgManagerLower, "yarn@") {
						result.PackageManagers = append(result.PackageManagers, "yarn")
					} else if strings.HasPrefix(pkgManagerLower, "pnpm@") {
						result.PackageManagers = append(result.PackageManagers, "pnpm")
					} else if strings.HasPrefix(pkgManagerLower, "bun@") {
						result.PackageManagers = append(result.PackageManagers, "bun")
					} else if strings.HasPrefix(pkgManagerLower, "npm@") {
						result.PackageManagers = append(result.PackageManagers, "npm")
					}
				}
			}
		}

		// If no packageManager field, check lock files (priority order)
		if !hasPackageManagerField {
			if a.fileExists("yarn.lock") {
				result.PackageManagers = append(result.PackageManagers, "yarn")
			} else if a.fileExists("pnpm-lock.yaml") {
				result.PackageManagers = append(result.PackageManagers, "pnpm")
			} else if a.fileExists("bun.lockb") {
				result.PackageManagers = append(result.PackageManagers, "bun")
			} else if a.fileExists("package-lock.json") {
				result.PackageManagers = append(result.PackageManagers, "npm")
			} else {
				// No lock file - check config files as hints
				if a.fileExists(".yarnrc") || a.fileExists(".yarnrc.yml") {
					result.PackageManagers = append(result.PackageManagers, "yarn")
				} else {
					// Default to npm if package.json exists but no indicators
					result.PackageManagers = append(result.PackageManagers, "npm")
				}
			}
		}

		// Parse package.json for dependencies
		if err := a.parsePackageJSON(result); err != nil {
			return err
		}
	}

	// Python package managers - check in priority order
	// Poetry (highest priority - has lock file)
	if a.fileExists("poetry.lock") {
		result.PackageManagers = append(result.PackageManagers, "poetry")
	} else if a.fileExists("pyproject.toml") {
		// Check if pyproject.toml contains poetry configuration
		data, err := os.ReadFile(filepath.Join(a.projectPath, "pyproject.toml"))
		if err == nil {
			content := string(data)
			if strings.Contains(content, "[tool.poetry]") {
				result.PackageManagers = append(result.PackageManagers, "poetry")
			}
		}
	}

	// Pipenv (check Pipfile.lock for definitive indicator)
	if a.fileExists("Pipfile.lock") {
		result.PackageManagers = append(result.PackageManagers, "pipenv")
	} else if a.fileExists("Pipfile") {
		result.PackageManagers = append(result.PackageManagers, "pipenv")
	}

	// pip (requirements.txt or setup.py)
	if a.fileExists("requirements.txt") {
		// Only add pip if not already using poetry or pipenv
		hasPoetryOrPipenv := false
		for _, pm := range result.PackageManagers {
			if pm == "poetry" || pm == "pipenv" {
				hasPoetryOrPipenv = true
				break
			}
		}
		if !hasPoetryOrPipenv {
			result.PackageManagers = append(result.PackageManagers, "pip")
		}
	} else if a.fileExists("setup.py") {
		// Only add pip if not already using poetry or pipenv
		hasPoetryOrPipenv := false
		for _, pm := range result.PackageManagers {
			if pm == "poetry" || pm == "pipenv" {
				hasPoetryOrPipenv = true
				break
			}
		}
		if !hasPoetryOrPipenv {
			result.PackageManagers = append(result.PackageManagers, "pip")
		}
	}

	// Go
	if a.fileExists("go.mod") {
		result.PackageManagers = append(result.PackageManagers, "go")
	}

	// Rust
	if a.fileExists("Cargo.toml") {
		result.PackageManagers = append(result.PackageManagers, "cargo")
	}

	// Java
	if a.fileExists("pom.xml") {
		result.PackageManagers = append(result.PackageManagers, "maven")
	}
	if a.fileExists("build.gradle") || a.fileExists("build.gradle.kts") {
		result.PackageManagers = append(result.PackageManagers, "gradle")
	}

	return nil
}

// detectEntryPoints finds main entry points
func (a *Analyzer) detectEntryPoints(result *types.AnalysisResult) error {
	// Node.js entry points
	if a.fileExists("package.json") {
		data, err := os.ReadFile(filepath.Join(a.projectPath, "package.json"))
		if err == nil {
			var pkg map[string]interface{}
			if json.Unmarshal(data, &pkg) == nil {
				if main, ok := pkg["main"].(string); ok {
					result.EntryPoints = append(result.EntryPoints, main)
				}
			}
		}
	}

	// Python entry points
	commonPythonEntry := []string{"main.py", "app.py", "server.py", "manage.py"}
	for _, entry := range commonPythonEntry {
		if a.fileExists(entry) {
			result.EntryPoints = append(result.EntryPoints, entry)
		}
	}

	// Go entry points
	if a.fileExists("main.go") {
		result.EntryPoints = append(result.EntryPoints, "main.go")
	}

	return nil
}

// analyzeReadme extracts information from README
func (a *Analyzer) analyzeReadme(result *types.AnalysisResult) error {
	// Check for README files
	readmeFiles := []string{"README.md", "README.MD", "readme.md", "README.txt", "README"}

	for _, readmeFile := range readmeFiles {
		if a.fileExists(readmeFile) {
			// TODO: Parse README and extract description, commands, etc.
			// This will be enhanced with AI analysis
			data, err := os.ReadFile(filepath.Join(a.projectPath, readmeFile))
			if err == nil && len(data) > 0 {
				// Store README content for AI analysis
				result.Description = "Found README: " + readmeFile
			}
			break
		}
	}

	return nil
}

// generateCommands creates setup commands based on project type
func (a *Analyzer) generateCommands(result *types.AnalysisResult, detectedTypes []types.ProjectType) {
	commandID := 1

	for _, pType := range detectedTypes {
		switch pType {
		case types.ProjectTypeNodeJS:
			// Determine which package manager to use (use same logic as infrastructure analyzer)
			pkgManager := a.detectNodePackageManager()
			var installCmd string
			var runCmd string

			// Check if this is a Next.js project
			isNextJS := a.isNextJSProject()

			switch pkgManager {
			case "yarn":
				installCmd = "yarn install"
				if isNextJS {
					runCmd = "yarn dev"
				} else {
					runCmd = "yarn start"
				}
			case "pnpm":
				installCmd = "pnpm install"
				if isNextJS {
					runCmd = "pnpm dev"
				} else {
					runCmd = "pnpm start"
				}
			case "bun":
				installCmd = "bun install"
				if isNextJS {
					runCmd = "bun dev"
				} else {
					runCmd = "bun start"
				}
			default:
				installCmd = "npm install"
				if isNextJS {
					runCmd = "npm run dev"
				} else {
					runCmd = "npm start"
				}
			}

			result.Commands = append(result.Commands, types.Command{
				ID:          fmt.Sprintf("cmd-%d", commandID),
				Description: fmt.Sprintf("Install Node.js dependencies (%s)", pkgManager),
				Command:     installCmd,
				WorkingDir:  a.projectPath,
				Stage:       "setup",
				Required:    true,
				Status:      types.CommandPending,
			})
			commandID++

			// Add run command
			runCmd = ""
			switch pkgManager {
			case "yarn":
				runCmd = "yarn dev"
			case "pnpm":
				runCmd = "pnpm dev"
			case "bun":
				runCmd = "bun dev"
			default:
				runCmd = "npm run dev"
			}

			// Check if dev script exists, otherwise try start
			if a.hasScript("dev") {
				// runCmd is already set to dev
			} else if a.hasScript("start") {
				switch pkgManager {
				case "yarn":
					runCmd = "yarn start"
				case "pnpm":
					runCmd = "pnpm start"
				case "bun":
					runCmd = "bun start"
				default:
					runCmd = "npm start"
				}
			}

			result.Commands = append(result.Commands, types.Command{
				ID:          fmt.Sprintf("cmd-%d", commandID),
				Description: "Start Application",
				Command:     runCmd,
				WorkingDir:  a.projectPath,
				Stage:       "run",
				Required:    false, // Not required for setup
				Status:      types.CommandPending,
			})
			commandID++

			// Check if start script exists (basic check)
			// For now, we assume standard start command, but we could check package.json
			cmdDesc := "Start Application"
			if isNextJS {
				cmdDesc = "Start Application (Development Mode)"
			}
			result.Commands = append(result.Commands, types.Command{
				ID:          fmt.Sprintf("cmd-%d", commandID),
				Description: cmdDesc,
				Command:     runCmd,
				WorkingDir:  a.projectPath,
				Stage:       "run",
				Required:    false, // Not required for setup
				Status:      types.CommandPending,
			})
			commandID++

		case types.ProjectTypePython:
			// Determine which Python package manager to use
			pkgManager, installCmd := a.detectPythonPackageManager()

			result.Commands = append(result.Commands, types.Command{
				ID:          fmt.Sprintf("cmd-%d", commandID),
				Description: fmt.Sprintf("Install Python dependencies (%s)", pkgManager),
				Command:     installCmd,
				WorkingDir:  a.projectPath,
				Stage:       "setup",
				Required:    true,
				Status:      types.CommandPending,
			})
			commandID++

			// Attempt to detect start command
			var startCmd string
			if a.fileExists("main.py") {
				startCmd = "python main.py"
			} else if a.fileExists("app.py") {
				startCmd = "python app.py"
			} else if a.fileExists("manage.py") {
				startCmd = "python manage.py runserver"
			}

			if startCmd != "" {
				result.Commands = append(result.Commands, types.Command{
					ID:          fmt.Sprintf("cmd-%d", commandID),
					Description: "Start Application",
					Command:     startCmd,
					WorkingDir:  a.projectPath,
					Stage:       "run",
					Required:    false,
					Status:      types.CommandPending,
				})
				commandID++
			}

		case types.ProjectTypeGo:
			result.Commands = append(result.Commands, types.Command{
				ID:          fmt.Sprintf("cmd-%d", commandID),
				Description: "Download Go dependencies",
				Command:     "go mod download",
				WorkingDir:  a.projectPath,
				Stage:       "setup",
				Required:    true,
				Status:      types.CommandPending,
			})
			commandID++

			if a.fileExists("main.go") {
				result.Commands = append(result.Commands, types.Command{
					ID:          fmt.Sprintf("cmd-%d", commandID),
					Description: "Run Application",
					Command:     "go run main.go",
					WorkingDir:  a.projectPath,
					Stage:       "run",
					Required:    false,
					Status:      types.CommandPending,
				})
				commandID++
			} else {
				// Check for cmd/ directory
				cmdDir := filepath.Join(a.projectPath, "cmd")
				if entries, err := os.ReadDir(cmdDir); err == nil {
					for _, entry := range entries {
						if entry.IsDir() {
							mainFile := filepath.Join("cmd", entry.Name(), "main.go")
							if a.fileExists(mainFile) {
								result.Commands = append(result.Commands, types.Command{
									ID:          fmt.Sprintf("cmd-%d", commandID),
									Description: "Run Application",
									Command:     fmt.Sprintf("go run %s", mainFile),
									WorkingDir:  a.projectPath,
									Stage:       "run",
									Required:    false,
									Status:      types.CommandPending,
								})
								commandID++
								break
							}
						}
					}
				}
			}

			if a.fileExists("main.go") {
				result.Commands = append(result.Commands, types.Command{
					ID:          fmt.Sprintf("cmd-%d", commandID),
					Description: "Run Application",
					Command:     "go run main.go",
					WorkingDir:  a.projectPath,
					Stage:       "run",
					Required:    false,
					Status:      types.CommandPending,
				})
				commandID++
			}
		}
	}
}

// parsePackageJSON extracts dependencies from package.json
func (a *Analyzer) parsePackageJSON(result *types.AnalysisResult) error {
	data, err := os.ReadFile(filepath.Join(a.projectPath, "package.json"))
	if err != nil {
		return err
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return err
	}

	// Extract dependencies
	if deps, ok := pkg["dependencies"].(map[string]interface{}); ok {
		for name, version := range deps {
			result.Dependencies = append(result.Dependencies, types.Dependency{
				Name:      name,
				Version:   fmt.Sprintf("%v", version),
				Type:      "package",
				Installed: false,
			})
		}
	}

	return nil
}

// estimateSize estimates project size
func (a *Analyzer) estimateSize() string {
	// TODO: Calculate actual size
	return "unknown"
}

// fileExists checks if a file exists in the project directory
func (a *Analyzer) fileExists(filename string) bool {
	path := filepath.Join(a.projectPath, filename)
	_, err := os.Stat(path)
	return err == nil
}

// detectNodePackageManager determines the Node.js package manager to use
func (a *Analyzer) detectNodePackageManager() string {
	// Check package.json for packageManager field (Node.js 16.9+) - most authoritative
	if a.fileExists("package.json") {
		data, err := os.ReadFile(filepath.Join(a.projectPath, "package.json"))
		if err == nil {
			var pkg map[string]interface{}
			if json.Unmarshal(data, &pkg) == nil {
				if pkgManager, ok := pkg["packageManager"].(string); ok {
					// Format: "npm@8.1.0", "yarn@3.2.0", "pnpm@7.0.0", "bun@1.0.0"
					pkgManagerLower := strings.ToLower(pkgManager)
					if strings.HasPrefix(pkgManagerLower, "yarn@") {
						return "yarn"
					}
					if strings.HasPrefix(pkgManagerLower, "pnpm@") {
						return "pnpm"
					}
					if strings.HasPrefix(pkgManagerLower, "bun@") {
						return "bun"
					}
					if strings.HasPrefix(pkgManagerLower, "npm@") {
						return "npm"
					}
				}
			}
		}
	}

	// Check for lock files (priority order: yarn > pnpm > bun > npm)
	if a.fileExists("yarn.lock") {
		return "yarn"
	}
	if a.fileExists("pnpm-lock.yaml") {
		return "pnpm"
	}
	if a.fileExists("bun.lockb") {
		return "bun"
	}
	if a.fileExists("package-lock.json") {
		return "npm"
	}

	// Check for config files as hints
	if a.fileExists(".yarnrc") || a.fileExists(".yarnrc.yml") {
		return "yarn"
	}

	// Default to npm if package.json exists
	if a.fileExists("package.json") {
		return "npm"
	}

	return "npm"
}

// detectPythonPackageManager determines the Python package manager to use
func (a *Analyzer) detectPythonPackageManager() (string, string) {
	// Priority: poetry > pipenv > pip
	// poetry.lock is the most definitive indicator
	if a.fileExists("poetry.lock") {
		return "poetry", "poetry install"
	}

	// Check for pyproject.toml with poetry section
	if a.fileExists("pyproject.toml") {
		data, err := os.ReadFile(filepath.Join(a.projectPath, "pyproject.toml"))
		if err == nil {
			content := string(data)
			// Check for poetry configuration
			if strings.Contains(content, "[tool.poetry]") {
				return "poetry", "poetry install"
			}
			// Check for other build systems that use pyproject.toml
			if strings.Contains(content, "[build-system]") {
				// Check if it's using poetry as build backend
				if strings.Contains(content, "poetry-core") || strings.Contains(content, "poetry.masonry") {
					return "poetry", "poetry install"
				}
			}
		}
	}

	// Check for Pipenv (Pipfile.lock is definitive)
	if a.fileExists("Pipfile.lock") {
		return "pipenv", "pipenv install"
	}
	// Pipfile indicates pipenv (even without lock)
	if a.fileExists("Pipfile") {
		return "pipenv", "pipenv install"
	}

	// Check for requirements.txt (pip)
	if a.fileExists("requirements.txt") {
		return "pip", "pip install -r requirements.txt"
	}

	// Check for setup.py (legacy setuptools)
	if a.fileExists("setup.py") {
		return "pip", "pip install -e ."
	}

	// Check for setup.cfg (setuptools configuration)
	if a.fileExists("setup.cfg") {
		return "pip", "pip install -e ."
	}

	// Default fallback
	return "pip", "pip install -r requirements.txt"
}

// isNextJSProject checks if the project is a Next.js project
func (a *Analyzer) isNextJSProject() bool {
	packageJSONPath := filepath.Join(a.projectPath, "package.json")
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return false
	}

	var pkgJSON map[string]interface{}
	if err := json.Unmarshal(data, &pkgJSON); err != nil {
		return false
	}

	// Check dependencies for "next"
	if deps, ok := pkgJSON["dependencies"].(map[string]interface{}); ok {
		if _, hasNext := deps["next"]; hasNext {
			return true
		}
	}

	// Check devDependencies for "next"
	if devDeps, ok := pkgJSON["devDependencies"].(map[string]interface{}); ok {
		if _, hasNext := devDeps["next"]; hasNext {
			return true
		}
	}

	return false
}

// hasScript checks if a script exists in package.json
func (a *Analyzer) hasScript(scriptName string) bool {
	if !a.fileExists("package.json") {
		return false
	}

	data, err := os.ReadFile(filepath.Join(a.projectPath, "package.json"))
	if err != nil {
		return false
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}

	if scripts, ok := pkg["scripts"].(map[string]interface{}); ok {
		_, exists := scripts[scriptName]
		return exists
	}

	return false
}
