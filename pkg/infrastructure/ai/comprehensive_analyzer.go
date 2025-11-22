package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ComprehensiveAIAnalyzer performs deep project analysis
type ComprehensiveAIAnalyzer struct {
	client          *Client
	fileAnalyzer    *FileAnalyzer
	depGraphBuilder *DependencyGraphBuilder
	riskAssessor    *AnalyzerRiskAssessor
}

// Technology represents a detected technology
type Technology struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"` // language, framework, tool
}

// VersionConflict represents a dependency version conflict
type VersionConflict struct {
	Dependency string   `json:"dependency"`
	Versions   []string `json:"versions"`
	Severity   string   `json:"severity"` // high, medium, low
}

// ComprehensiveAnalysis contains the complete AI analysis
type ComprehensiveAnalysis struct {
	// Project Understanding
	ProjectType    string       `json:"project_type"`
	Technologies   []Technology `json:"technologies"`
	Framework      string       `json:"framework"`
	BuildSystem    string       `json:"build_system"`
	PackageManager string       `json:"package_manager"`

	// Dependency Analysis
	DependencyGraph  *DependencyGraph   `json:"dependency_graph"`
	MissingDeps      []string           `json:"missing_deps"`
	VersionConflicts []VersionConflict  `json:"version_conflicts"`

	// Execution Plan
	ExecutionPlan  *ExecutionPlan  `json:"execution_plan"`
	EstimatedTime  time.Duration   `json:"estimated_time"`
	RiskAssessment *RiskAssessment `json:"risk_assessment"`

	// Monitoring Strategy
	MonitoringPlan      *MonitoringPlan `json:"monitoring_plan"`
	FallbackStrategies  []FallbackStrategy `json:"fallback_strategies"`

	// Confidence
	Confidence  float64 `json:"confidence"`
	Explanation string  `json:"explanation"`
}

// DependencyGraph represents the dependency structure
type DependencyGraph struct {
	Root      string            `json:"root"`
	Nodes     []DependencyNode  `json:"nodes"`
	Edges     []DependencyEdge  `json:"edges"`
}

// DependencyNode represents a dependency
type DependencyNode struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

// DependencyEdge represents a dependency relationship
type DependencyEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"` // direct, transitive
}

// ExecutionPlan contains the execution strategy
type ExecutionPlan struct {
	Phases   []ExecutionPhase `json:"phases"`
	Rollback *RollbackPlan    `json:"rollback"`
}

// ExecutionPhase represents a phase of execution
type ExecutionPhase struct {
	Name     string    `json:"name"`
	Commands []Command `json:"commands"`
	Reason   string    `json:"reason"`
	CanFail  bool      `json:"can_fail"`
	Recovery *RecoveryStrategy `json:"recovery"`
	Timeout  time.Duration `json:"timeout"`
}

// Command represents a command in the execution plan
type Command struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Command     string `json:"command"`
	WorkingDir  string `json:"working_dir"`
	Stage       string `json:"stage"`
	Required    bool   `json:"required"`
}

// RecoveryStrategy defines how to recover from failures
type RecoveryStrategy struct {
	Retry         bool   `json:"retry"`
	MaxAttempts   int    `json:"max_attempts"`
	FallbackCommand string `json:"fallback_command"`
	Backoff       string `json:"backoff"` // linear, exponential
}

// RollbackPlan defines how to rollback changes
type RollbackPlan struct {
	Steps []RollbackStep `json:"steps"`
}

// RollbackStep represents a rollback action
type RollbackStep struct {
	Action string `json:"action"`
	Command string `json:"command"`
}

// AnalyzerRiskAssessment evaluates project risks
type AnalyzerRiskAssessment struct {
	OverallRisk   string   `json:"overall_risk"` // high, medium, low
	RiskFactors   []string `json:"risk_factors"`
	Mitigations   []string `json:"mitigations"`
}

// MonitoringPlan defines what to monitor during execution
type MonitoringPlan struct {
	WatchFor      []string       `json:"watch_for"`      // Error patterns
	AlertOn       []string       `json:"alert_on"`       // Conditions to alert
	RecoveryRules []RecoveryRule `json:"recovery_rules"`
}

// RecoveryRule defines when to trigger recovery
type RecoveryRule struct {
	Pattern   string `json:"pattern"`
	Action   string `json:"action"`
	Severity string `json:"severity"`
}

// FallbackStrategy defines fallback behavior
type FallbackStrategy struct {
	Trigger    string  `json:"trigger"`
	Strategy   string  `json:"strategy"`
	Confidence float64 `json:"confidence"`
}

// FileAnalyzer analyzes project files
type FileAnalyzer struct{}

// DependencyGraphBuilder builds dependency graphs
type DependencyGraphBuilder struct{}

// AnalyzerRiskAssessor assesses project risks
type AnalyzerRiskAssessor struct{}

// NewComprehensiveAIAnalyzer creates a new comprehensive analyzer
func NewComprehensiveAIAnalyzer(client *Client) *ComprehensiveAIAnalyzer {
	return &ComprehensiveAIAnalyzer{
		client:          client,
		fileAnalyzer:    &FileAnalyzer{},
		depGraphBuilder: &DependencyGraphBuilder{},
		riskAssessor:    &AnalyzerRiskAssessor{},
	}
}

// Analyze performs comprehensive analysis of a project
func (a *ComprehensiveAIAnalyzer) Analyze(ctx context.Context, projectPath string) (*ComprehensiveAnalysis, error) {
	// Read project files
	readmeContent := a.readReadme(projectPath)
	fileList := a.getFileList(projectPath)
	packageConfig := a.readPackageConfig(projectPath)
	buildConfig := a.readBuildConfig(projectPath)

	// Build comprehensive prompt
	prompt := a.buildComprehensivePrompt(projectPath, readmeContent, fileList, packageConfig, buildConfig)

	// Call AI with retry logic
	messages := []Message{
		{
			Role:    "system",
			Content: "You are a senior DevOps engineer analyzing a project for automated setup. Provide comprehensive analysis and execution plan in JSON format.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Retry with exponential backoff
	maxAttempts := 3
	initialDelay := 1 * time.Second
	var response string
	var err error
	
	for attempt := 0; attempt < maxAttempts; attempt++ {
		response, err = a.client.Chat(messages)
		if err == nil {
			break
		}
		
		// Don't retry on last attempt
		if attempt < maxAttempts-1 {
			delay := time.Duration(float64(initialDelay) * math.Pow(2, float64(attempt)))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		}
	}
	
	if err != nil {
		return nil, fmt.Errorf("AI analysis failed after %d attempts: %w", maxAttempts, err)
	}

	// Parse response
	analysis, err := a.parseAnalysisResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse analysis: %w", err)
	}

	return analysis, nil
}

// buildComprehensivePrompt creates the comprehensive analysis prompt
func (a *ComprehensiveAIAnalyzer) buildComprehensivePrompt(projectPath, readmeContent string, fileList []string, packageConfig, buildConfig string) string {
	return fmt.Sprintf(`You are a senior DevOps engineer analyzing a project for automated setup.

PROJECT FILES:
%s

KEY FILES CONTENT:
README:
%s

PACKAGE CONFIG:
%s

BUILD CONFIG:
%s

TASK: Provide a comprehensive analysis and execution plan.

ANALYSIS FRAMEWORK:
1. Project Understanding
   - What is this project? (web app, CLI, library, etc.)
   - Primary language and framework
   - Build system and package manager
   - Runtime requirements

2. Dependency Analysis
   - Map all dependencies
   - Identify missing dependencies
   - Detect version conflicts
   - Check for security vulnerabilities

3. Execution Strategy
   - Optimal command sequence
   - Parallel vs. sequential execution
   - Risk points and mitigation
   - Expected time for each phase

4. Monitoring Strategy
   - Critical error patterns to watch
   - Success criteria for each command
   - Recovery strategies for common failures

5. Confidence Assessment
   - How confident are you? (0.0-1.0)
   - What are the risk factors?
   - What could go wrong?

OUTPUT (JSON):
{
  "project_type": "...",
  "technologies": [{"name": "...", "version": "...", "type": "..."}],
  "framework": "...",
  "build_system": "...",
  "package_manager": "...",
  "dependency_graph": {
    "root": "...",
    "nodes": [{"name": "...", "version": "...", "type": "...", "required": true}],
    "edges": [{"from": "...", "to": "...", "type": "direct"}]
  },
  "missing_deps": ["..."],
  "version_conflicts": [{"dependency": "...", "versions": ["..."], "severity": "high"}],
  "execution_plan": {
    "phases": [
      {
        "name": "Install Dependencies",
        "commands": [{"id": "cmd-1", "description": "...", "command": "...", "working_dir": "%s", "stage": "setup", "required": true}],
        "reason": "Install packages from package.json",
        "can_fail": false,
        "recovery": {
          "retry": true,
          "max_attempts": 3,
          "fallback_command": "npm ci",
          "backoff": "exponential"
        },
        "timeout": "5m"
      }
    ],
    "rollback": {
      "steps": [{"action": "remove", "command": "rm -rf node_modules"}]
    }
  },
  "estimated_time": "10m",
  "risk_assessment": {
    "overall_risk": "low",
    "risk_factors": ["..."],
    "mitigations": ["..."]
  },
  "monitoring_plan": {
    "watch_for": ["Cannot find module", "EACCES", "ENOENT"],
    "alert_on": ["Build failed", "Tests failed"],
    "recovery_rules": [{"pattern": "Cannot find module", "action": "retry", "severity": "medium"}]
  },
  "fallback_strategies": [{"trigger": "low_confidence", "strategy": "traditional", "confidence": 0.5}],
  "confidence": 0.92,
  "explanation": "High confidence because package.json is well-formed and dependencies are standard."
}

RULES:
1. Set "working_dir" to "%s" for ALL commands (no cd commands)
2. Each command must be directly executable (executable + args only)
3. Use the correct interpreter for each file type based on its extension
4. Generate commands appropriate for Windows
5. Think step-by-step: what would a developer need to do to run this project?

Provide ONLY valid JSON.`,
		strings.Join(fileList, "\n"),
		readmeContent,
		packageConfig,
		buildConfig,
		projectPath,
		projectPath)
}

// parseAnalysisResponse parses the AI response into ComprehensiveAnalysis
func (a *ComprehensiveAIAnalyzer) parseAnalysisResponse(response string) (*ComprehensiveAnalysis, error) {
	// Extract JSON from response
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := response[start : end+1]

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	analysis := &ComprehensiveAnalysis{
		Confidence: 0.7, // Default confidence
	}

	// Parse basic fields
	if pt, ok := raw["project_type"].(string); ok {
		analysis.ProjectType = pt
	}
	if fw, ok := raw["framework"].(string); ok {
		analysis.Framework = fw
	}
	if bs, ok := raw["build_system"].(string); ok {
		analysis.BuildSystem = bs
	}
	if pm, ok := raw["package_manager"].(string); ok {
		analysis.PackageManager = pm
	}
	if conf, ok := raw["confidence"].(float64); ok {
		analysis.Confidence = conf
	}
	if exp, ok := raw["explanation"].(string); ok {
		analysis.Explanation = exp
	}

	// Parse technologies
	if techs, ok := raw["technologies"].([]interface{}); ok {
		for _, t := range techs {
			if techMap, ok := t.(map[string]interface{}); ok {
				tech := Technology{}
				if name, ok := techMap["name"].(string); ok {
					tech.Name = name
				}
				if version, ok := techMap["version"].(string); ok {
					tech.Version = version
				}
				if typ, ok := techMap["type"].(string); ok {
					tech.Type = typ
				}
				analysis.Technologies = append(analysis.Technologies, tech)
			}
		}
	}

	// Parse execution plan
	if ep, ok := raw["execution_plan"].(map[string]interface{}); ok {
		analysis.ExecutionPlan = &ExecutionPlan{}
		
		if phases, ok := ep["phases"].([]interface{}); ok {
			for _, p := range phases {
				if phaseMap, ok := p.(map[string]interface{}); ok {
					phase := ExecutionPhase{}
					if name, ok := phaseMap["name"].(string); ok {
						phase.Name = name
					}
					if reason, ok := phaseMap["reason"].(string); ok {
						phase.Reason = reason
					}
					if canFail, ok := phaseMap["can_fail"].(bool); ok {
						phase.CanFail = canFail
					}
					
					// Parse commands
					if cmds, ok := phaseMap["commands"].([]interface{}); ok {
						for _, c := range cmds {
							if cmdMap, ok := c.(map[string]interface{}); ok {
								cmd := Command{}
								if id, ok := cmdMap["id"].(string); ok {
									cmd.ID = id
								}
								if desc, ok := cmdMap["description"].(string); ok {
									cmd.Description = desc
								}
								if cmdStr, ok := cmdMap["command"].(string); ok {
									cmd.Command = cmdStr
								}
								if wd, ok := cmdMap["working_dir"].(string); ok {
									cmd.WorkingDir = wd
								}
								if stage, ok := cmdMap["stage"].(string); ok {
									cmd.Stage = stage
								}
								if req, ok := cmdMap["required"].(bool); ok {
									cmd.Required = req
								}
								phase.Commands = append(phase.Commands, cmd)
							}
						}
					}
					
					analysis.ExecutionPlan.Phases = append(analysis.ExecutionPlan.Phases, phase)
				}
			}
		}
	}

	// Parse monitoring plan
	if mp, ok := raw["monitoring_plan"].(map[string]interface{}); ok {
		analysis.MonitoringPlan = &MonitoringPlan{}
		
		if watchFor, ok := mp["watch_for"].([]interface{}); ok {
			for _, w := range watchFor {
				if str, ok := w.(string); ok {
					analysis.MonitoringPlan.WatchFor = append(analysis.MonitoringPlan.WatchFor, str)
				}
			}
		}
	}

	// Parse missing deps
	if md, ok := raw["missing_deps"].([]interface{}); ok {
		for _, d := range md {
			if str, ok := d.(string); ok {
				analysis.MissingDeps = append(analysis.MissingDeps, str)
			}
		}
	}

	return analysis, nil
}

// readReadme finds and reads the README file
func (a *ComprehensiveAIAnalyzer) readReadme(projectPath string) string {
	readmeFiles := []string{"README.md", "README.MD", "readme.md", "README.txt", "README", "Readme.md"}
	for _, readmeFile := range readmeFiles {
		path := filepath.Join(projectPath, readmeFile)
		if data, err := os.ReadFile(path); err == nil {
			return string(data)
		}
	}
	return ""
}

// getFileList returns a list of files in the project
func (a *ComprehensiveAIAnalyzer) getFileList(projectPath string) []string {
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

// readPackageConfig reads package configuration files
func (a *ComprehensiveAIAnalyzer) readPackageConfig(projectPath string) string {
	configFiles := []string{"package.json", "requirements.txt", "go.mod", "Cargo.toml", "pom.xml", "build.gradle"}
	for _, configFile := range configFiles {
		path := filepath.Join(projectPath, configFile)
		if data, err := os.ReadFile(path); err == nil {
			return string(data)
		}
	}
	return ""
}

// readBuildConfig reads build configuration files
func (a *ComprehensiveAIAnalyzer) readBuildConfig(projectPath string) string {
	buildFiles := []string{"Makefile", "build.sh", "build.bat", "CMakeLists.txt", "tsconfig.json", "webpack.config.js"}
	for _, buildFile := range buildFiles {
		path := filepath.Join(projectPath, buildFile)
		if data, err := os.ReadFile(path); err == nil {
			return string(data)
		}
	}
	return ""
}

