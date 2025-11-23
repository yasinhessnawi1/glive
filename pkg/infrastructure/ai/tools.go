package ai

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ToolType represents the type of tool
type ToolType string

const (
	ToolRunCommand ToolType = "run_command"
	ToolReadFile   ToolType = "read_file"
	ToolListDir    ToolType = "list_dir"
	ToolReadDocs   ToolType = "read_docs"
	ToolEditFile   ToolType = "edit_file"
)

// ToolCall represents a request to execute a tool
type ToolCall struct {
	Type      ToolType          `json:"type"`
	Arguments map[string]string `json:"arguments"`
}

// ToolResult represents the output of a tool execution
type ToolResult struct {
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
	Success bool   `json:"success"`
}

// ToolExecutor handles tool execution
type ToolExecutor struct {
	workingDir string
}

// NewToolExecutor creates a new tool executor
func NewToolExecutor(workingDir string) *ToolExecutor {
	return &ToolExecutor{
		workingDir: workingDir,
	}
}

// Execute runs a tool
func (e *ToolExecutor) Execute(ctx context.Context, call ToolCall) *ToolResult {
	switch call.Type {
	case ToolRunCommand:
		return e.runCommand(ctx, call.Arguments["command"])
	case ToolReadFile:
		return e.readFile(call.Arguments["path"])
	case ToolListDir:
		return e.listDir(call.Arguments["path"])
	case ToolReadDocs:
		return e.readDocs(call.Arguments["path"])
	case ToolEditFile:
		return e.editFile(call.Arguments["path"], call.Arguments["content"])
	default:
		return &ToolResult{
			Success: false,
			Error:   fmt.Sprintf("unknown tool: %s", call.Type),
		}
	}
}

func (e *ToolExecutor) runCommand(ctx context.Context, cmdStr string) *ToolResult {
	if cmdStr == "" {
		return &ToolResult{Success: false, Error: "command is empty"}
	}

	// Create command
	// Use shell to allow piping and environment variables
	var cmd *exec.Cmd
	if strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") {
		cmd = exec.CommandContext(ctx, "cmd", "/C", cmdStr)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", cmdStr)
	}

	cmd.Dir = e.workingDir

	// Capture output
	output, err := cmd.CombinedOutput()

	result := &ToolResult{
		Output:  string(output),
		Success: err == nil,
	}

	if err != nil {
		result.Error = err.Error()
	}

	return result
}

func (e *ToolExecutor) readFile(path string) *ToolResult {
	if path == "" {
		return &ToolResult{Success: false, Error: "path is empty"}
	}

	// Resolve absolute path
	fullPath := path
	if !filepath.IsAbs(path) {
		fullPath = filepath.Join(e.workingDir, path)
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}
	}

	// Truncate if too large (limit to ~10KB for now)
	if len(content) > 10000 {
		return &ToolResult{
			Success: true,
			Output:  string(content[:10000]) + "\n... (truncated)",
		}
	}

	return &ToolResult{
		Success: true,
		Output:  string(content),
	}
}

func (e *ToolExecutor) listDir(path string) *ToolResult {
	targetDir := e.workingDir
	if path != "" {
		if filepath.IsAbs(path) {
			targetDir = path
		} else {
			targetDir = filepath.Join(e.workingDir, path)
		}
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Contents of %s:\n", targetDir))

	for _, entry := range entries {
		prefix := "📄 "
		if entry.IsDir() {
			prefix = "📂 "
		}
		output.WriteString(fmt.Sprintf("%s%s\n", prefix, entry.Name()))
	}

	return &ToolResult{
		Success: true,
		Output:  output.String(),
	}
}

func (e *ToolExecutor) readDocs(path string) *ToolResult {
	// Find all markdown files in the repository
	targetDir := e.workingDir
	if path != "" {
		if filepath.IsAbs(path) {
			targetDir = path
		} else {
			targetDir = filepath.Join(e.workingDir, path)
		}
	}

	var docs strings.Builder
	docs.WriteString("Documentation found:\n\n")

	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip hidden directories
		if strings.HasPrefix(info.Name(), ".") && info.IsDir() {
			return filepath.SkipDir
		}

		// Skip node_modules and vendor
		if (info.Name() == "node_modules" || info.Name() == "vendor") && info.IsDir() {
			return filepath.SkipDir
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			content, err := os.ReadFile(path)
			if err == nil {
				relPath, _ := filepath.Rel(targetDir, path)
				docs.WriteString(fmt.Sprintf("--- %s ---\n", relPath))

				// Truncate individual doc files
				contentStr := string(content)
				if len(contentStr) > 2000 {
					contentStr = contentStr[:2000] + "\n... (truncated)"
				}
				docs.WriteString(contentStr)
				docs.WriteString("\n\n")
			}
		}
		return nil
	})

	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}
	}

	return &ToolResult{
		Success: true,
		Output:  docs.String(),
	}
}

func (e *ToolExecutor) editFile(path, content string) *ToolResult {
	if path == "" {
		return &ToolResult{Success: false, Error: "path is empty"}
	}

	// Resolve absolute path
	fullPath := path
	if !filepath.IsAbs(path) {
		fullPath = filepath.Join(e.workingDir, path)
	}

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &ToolResult{Success: false, Error: fmt.Sprintf("failed to create directory: %v", err)}
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return &ToolResult{Success: false, Error: err.Error()}
	}

	return &ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Successfully wrote to %s", path),
	}
}
