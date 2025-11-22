package analyzer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Analyzer analyzes project structure and determines setup requirements
type Analyzer struct {
	projectPath string
}

// ProjectType represents the detected project type
type ProjectType string

const (
	ProjectTypeNodeJS   ProjectType = "nodejs"
	ProjectTypePython   ProjectType = "python"
	ProjectTypeGo       ProjectType = "go"
	ProjectTypeRust     ProjectType = "rust"
	ProjectTypeJava     ProjectType = "java"
	ProjectTypeDocker   ProjectType = "docker"
	ProjectTypeUnknown  ProjectType = "unknown"
	ProjectTypePolyglot ProjectType = "polyglot"
)

// AnalysisResult contains the analysis of a project
type AnalysisResult struct {
	ProjectType        ProjectType
	DetectedLanguages  []string
	PackageManagers    []string
	EntryPoints        []string
	Dependencies       []Dependency
	SystemRequirements []string
	Commands           []Command
	IsSuspicious       bool
	SuspiciousReasons  []string
	EstimatedSize      string
	Description        string
	UsageInstructions  []string
	KeyMilestones      []string
}

// Dependency represents a required dependency
type Dependency struct {
	Name      string
	Version   string
	Type      string // system, language, package
	Installed bool
}

// Command represents a command to be executed
type Command struct {
	ID          string
	Description string
	Command     string
	WorkingDir  string
	Stage       string // setup, build, run
	Required    bool
	Status      string
}

// New creates a new Analyzer instance
func New(projectPath string) *Analyzer {
	return &Analyzer{
		projectPath: projectPath,
	}
}

// Analyze performs project analysis
func (a *Analyzer) Analyze() (*AnalysisResult, error) {
	result := &AnalysisResult{
		DetectedLanguages:  []string{},
		PackageManagers:    []string{},
		EntryPoints:        []string{},
		Dependencies:       []Dependency{},
		SystemRequirements: []string{},
		Commands:           []Command{},
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
func (a *Analyzer) detectProjectType() (ProjectType, []ProjectType) {
	detectedTypes := []ProjectType{}

	// Check for Node.js
	if a.fileExists("package.json") {
		detectedTypes = append(detectedTypes, ProjectTypeNodeJS)
	}

	// Check for Python
	if a.fileExists("requirements.txt") || a.fileExists("setup.py") ||
		a.fileExists("pyproject.toml") || a.fileExists("Pipfile") {
		detectedTypes = append(detectedTypes, ProjectTypePython)
	}

	// Check for Go
	if a.fileExists("go.mod") {
		detectedTypes = append(detectedTypes, ProjectTypeGo)
	}

	// Check for Rust
	if a.fileExists("Cargo.toml") {
		detectedTypes = append(detectedTypes, ProjectTypeRust)
	}

	// Check for Java
	if a.fileExists("pom.xml") || a.fileExists("build.gradle") {
		detectedTypes = append(detectedTypes, ProjectTypeJava)
	}

	// Check for Docker
	if a.fileExists("Dockerfile") || a.fileExists("docker-compose.yml") {
		detectedTypes = append(detectedTypes, ProjectTypeDocker)
	}

	// Determine primary type
	if len(detectedTypes) == 0 {
		return ProjectTypeUnknown, detectedTypes
	} else if len(detectedTypes) == 1 {
		return detectedTypes[0], detectedTypes
	} else {
		// Polyglot project - prioritize based on common patterns
		return ProjectTypePolyglot, detectedTypes
	}
}

// detectPackageManagers identifies package managers in use
func (a *Analyzer) detectPackageManagers(result *AnalysisResult) error {
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
		a.parsePyprojectToml(result)
	} else if a.fileExists("pyproject.toml") {
		// Check if pyproject.toml contains poetry configuration
		data, err := os.ReadFile(filepath.Join(a.projectPath, "pyproject.toml"))
		if err == nil {
			content := string(data)
			if strings.Contains(content, "[tool.poetry]") {
				result.PackageManagers = append(result.PackageManagers, "poetry")
				a.parsePyprojectToml(result)
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
		a.parseRequirementsTxt(result)
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
		a.parseGoMod(result)
	}

	// Rust
	if a.fileExists("Cargo.toml") {
		result.PackageManagers = append(result.PackageManagers, "cargo")
		a.parseCargoToml(result)
	}

	// Java - check both Maven and Gradle
	if a.fileExists("pom.xml") {
		result.PackageManagers = append(result.PackageManagers, "maven")
	}
	if a.fileExists("build.gradle") || a.fileExists("build.gradle.kts") {
		result.PackageManagers = append(result.PackageManagers, "gradle")
	}

	return nil
}

// detectEntryPoints finds main entry points
func (a *Analyzer) detectEntryPoints(result *AnalysisResult) error {
	// Node.js entry points
	if a.fileExists("package.json") {
		data, err := os.ReadFile(filepath.Join(a.projectPath, "package.json"))
		if err == nil {
			var pkg map[string]interface{}
			if json.Unmarshal(data, &pkg) == nil {
				if main, ok := pkg["main"].(string); ok {
					result.EntryPoints = append(result.EntryPoints, main)
				}
				// Check for scripts.start
				if scripts, ok := pkg["scripts"].(map[string]interface{}); ok {
					if _, ok := scripts["start"]; ok {
						result.EntryPoints = append(result.EntryPoints, "npm start")
					}
				}
			}
		}
	}

	// Python entry points - check common patterns
	pythonEntries := []string{
		"main.py", "app.py", "server.py", "manage.py", "run.py",
		"__main__.py", "cli.py", "wsgi.py", "asgi.py",
		"src/main.py", "src/__main__.py", "src/app.py",
	}
	for _, entry := range pythonEntries {
		if a.fileExists(entry) {
			result.EntryPoints = append(result.EntryPoints, entry)
		}
	}

	// Go entry points
	goEntries := []string{"main.go", "cmd/main.go"}
	for _, entry := range goEntries {
		if a.fileExists(entry) {
			result.EntryPoints = append(result.EntryPoints, entry)
		}
	}
	// Check cmd/ subdirectories
	cmdDir := filepath.Join(a.projectPath, "cmd")
	if entries, err := os.ReadDir(cmdDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				mainFile := filepath.Join("cmd", entry.Name(), "main.go")
				if a.fileExists(mainFile) {
					result.EntryPoints = append(result.EntryPoints, mainFile)
				}
			}
		}
	}

	// Rust entry points
	rustEntries := []string{"src/main.rs", "src/lib.rs"}
	for _, entry := range rustEntries {
		if a.fileExists(entry) {
			result.EntryPoints = append(result.EntryPoints, entry)
		}
	}

	// Java entry points
	javaEntries := []string{"src/main/java/Main.java", "src/Main.java", "Main.java"}
	for _, entry := range javaEntries {
		if a.fileExists(entry) {
			result.EntryPoints = append(result.EntryPoints, entry)
		}
	}

	// Docker entry points
	dockerEntries := []string{"Dockerfile", "docker-compose.yml", "docker-compose.yaml"}
	for _, entry := range dockerEntries {
		if a.fileExists(entry) {
			result.EntryPoints = append(result.EntryPoints, entry)
		}
	}

	return nil
}

// analyzeReadme extracts information from README
func (a *Analyzer) analyzeReadme(result *AnalysisResult) error {
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
func (a *Analyzer) generateCommands(result *AnalysisResult, detectedTypes []ProjectType) {
	commandID := 1

	for _, pType := range detectedTypes {
		switch pType {
		case ProjectTypeNodeJS:
			// Determine which package manager to use (prioritize lock files)
			pkgManager := a.detectNodePackageManager()
			var installCmd string

			// Use appropriate install command for each package manager
			switch pkgManager {
			case "yarn":
				installCmd = "yarn install"
			case "pnpm":
				installCmd = "pnpm install"
			case "bun":
				installCmd = "bun install"
			default:
				installCmd = "npm install"
			}

			result.Commands = append(result.Commands, Command{
				ID:          fmt.Sprintf("cmd-%d", commandID),
				Description: fmt.Sprintf("Install Node.js dependencies (%s)", pkgManager),
				Command:     installCmd,
				WorkingDir:  a.projectPath,
				Stage:       "setup",
				Required:    true,
				Status:      "pending",
			})
			commandID++

		case ProjectTypePython:
			// Determine which Python package manager to use
			pkgManager, installCmd := a.detectPythonPackageManager()

			result.Commands = append(result.Commands, Command{
				ID:          fmt.Sprintf("cmd-%d", commandID),
				Description: fmt.Sprintf("Install Python dependencies (%s)", pkgManager),
				Command:     installCmd,
				WorkingDir:  a.projectPath,
				Stage:       "setup",
				Required:    true,
				Status:      "pending",
			})
			commandID++

		case ProjectTypeGo:
			result.Commands = append(result.Commands, Command{
				ID:          fmt.Sprintf("cmd-%d", commandID),
				Description: "Download Go dependencies",
				Command:     "go mod download",
				WorkingDir:  a.projectPath,
				Stage:       "setup",
				Required:    true,
				Status:      "pending",
			})
			commandID++

		case ProjectTypeRust:
			result.Commands = append(result.Commands, Command{
				ID:          fmt.Sprintf("cmd-%d", commandID),
				Description: "Build Rust dependencies",
				Command:     "cargo build",
				WorkingDir:  a.projectPath,
				Stage:       "setup",
				Required:    true,
				Status:      "pending",
			})
			commandID++

		case ProjectTypeJava:
			// Determine which Java build tool to use
			if a.fileExists("pom.xml") {
				result.Commands = append(result.Commands, Command{
					ID:          fmt.Sprintf("cmd-%d", commandID),
					Description: "Download Maven dependencies",
					Command:     "mvn dependency:resolve",
					WorkingDir:  a.projectPath,
					Stage:       "setup",
					Required:    true,
					Status:      "pending",
				})
				commandID++
			} else if a.fileExists("build.gradle") || a.fileExists("build.gradle.kts") {
				result.Commands = append(result.Commands, Command{
					ID:          fmt.Sprintf("cmd-%d", commandID),
					Description: "Download Gradle dependencies",
					Command:     "gradle dependencies",
					WorkingDir:  a.projectPath,
					Stage:       "setup",
					Required:    true,
					Status:      "pending",
				})
				commandID++
			}
		}
	}
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
	// yarn.lock is most common and reliable
	if a.fileExists("yarn.lock") {
		return "yarn"
	}
	// pnpm-lock.yaml is definitive for pnpm
	if a.fileExists("pnpm-lock.yaml") {
		return "pnpm"
	}
	// bun.lockb is definitive for bun
	if a.fileExists("bun.lockb") {
		return "bun"
	}
	// package-lock.json indicates npm
	if a.fileExists("package-lock.json") {
		return "npm"
	}

	// Check for .npmrc or .yarnrc files as hints
	if a.fileExists(".yarnrc") || a.fileExists(".yarnrc.yml") {
		return "yarn"
	}
	if a.fileExists(".npmrc") {
		// Check .npmrc for pnpm-specific settings
		data, err := os.ReadFile(filepath.Join(a.projectPath, ".npmrc"))
		if err == nil {
			content := string(data)
			if strings.Contains(content, "shamefully-hoist") || strings.Contains(content, "strict-peer-dependencies") {
				// Common pnpm settings
				return "pnpm"
			}
		}
		return "npm"
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

	// Default fallback - try pip with requirements.txt (might not exist, but common pattern)
	return "pip", "pip install -r requirements.txt"
}

// parsePackageJSON extracts dependencies from package.json
func (a *Analyzer) parsePackageJSON(result *AnalysisResult) error {
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
			result.Dependencies = append(result.Dependencies, Dependency{
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

// parseRequirementsTxt parses Python requirements.txt for dependencies
func (a *Analyzer) parseRequirementsTxt(result *AnalysisResult) {
	file, err := os.Open(filepath.Join(a.projectPath, "requirements.txt"))
	if err != nil {
		return
	}
	defer file.Close()

	// Regex to parse requirements: package==version, package>=version, package, etc.
	depRegex := regexp.MustCompile(`^([a-zA-Z0-9_-]+)(?:\[.*\])?(?:([<>=!~]+)(.+))?`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}

		matches := depRegex.FindStringSubmatch(line)
		if len(matches) >= 2 {
			name := matches[1]
			version := ""
			if len(matches) >= 4 && matches[3] != "" {
				version = matches[2] + matches[3]
			}
			result.Dependencies = append(result.Dependencies, Dependency{
				Name:      name,
				Version:   version,
				Type:      "pip",
				Installed: false,
			})
		}
	}
}

// parsePyprojectToml parses pyproject.toml for dependencies
func (a *Analyzer) parsePyprojectToml(result *AnalysisResult) {
	data, err := os.ReadFile(filepath.Join(a.projectPath, "pyproject.toml"))
	if err != nil {
		return
	}

	content := string(data)
	// Simple regex to find dependencies in pyproject.toml
	depRegex := regexp.MustCompile(`["']([a-zA-Z0-9_-]+)(?:\[.*\])?(?:[<>=!~]+[^"']+)?["']`)

	// Look for dependencies section
	if strings.Contains(content, "dependencies") {
		matches := depRegex.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				result.Dependencies = append(result.Dependencies, Dependency{
					Name:      match[1],
					Version:   "",
					Type:      "poetry",
					Installed: false,
				})
			}
		}
	}
}

// parseGoMod parses go.mod for dependencies
func (a *Analyzer) parseGoMod(result *AnalysisResult) {
	file, err := os.Open(filepath.Join(a.projectPath, "go.mod"))
	if err != nil {
		return
	}
	defer file.Close()

	// Regex to parse require statements
	requireRegex := regexp.MustCompile(`^\s*([a-zA-Z0-9./_-]+)\s+v?([0-9.]+.*)`)

	scanner := bufio.NewScanner(file)
	inRequire := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "require (") {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}

		// Single require or inside require block
		if strings.HasPrefix(line, "require ") || inRequire {
			checkLine := line
			if strings.HasPrefix(line, "require ") {
				checkLine = strings.TrimPrefix(line, "require ")
			}

			matches := requireRegex.FindStringSubmatch(checkLine)
			if len(matches) >= 3 {
				// Skip indirect dependencies
				if strings.Contains(line, "// indirect") {
					continue
				}
				result.Dependencies = append(result.Dependencies, Dependency{
					Name:      matches[1],
					Version:   matches[2],
					Type:      "go",
					Installed: false,
				})
			}
		}
	}
}

// parseCargoToml parses Cargo.toml for dependencies
func (a *Analyzer) parseCargoToml(result *AnalysisResult) {
	data, err := os.ReadFile(filepath.Join(a.projectPath, "Cargo.toml"))
	if err != nil {
		return
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	// Simple parsing for [dependencies] section
	inDeps := false
	depRegex := regexp.MustCompile(`^([a-zA-Z0-9_-]+)\s*=\s*["']?([^"'\s]+)["']?`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "[dependencies]" || line == "[dev-dependencies]" {
			inDeps = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inDeps = false
			continue
		}

		if inDeps && line != "" && !strings.HasPrefix(line, "#") {
			matches := depRegex.FindStringSubmatch(line)
			if len(matches) >= 3 {
				result.Dependencies = append(result.Dependencies, Dependency{
					Name:      matches[1],
					Version:   matches[2],
					Type:      "cargo",
					Installed: false,
				})
			}
		}
	}
}
