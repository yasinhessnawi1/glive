package values

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/glive/domain/errors"
)

// ValidatedCommand is a command that has passed validation
type ValidatedCommand struct {
	executable string
	args       []string
	original   string
	workingDir string
}

// CommandValidator validates commands before execution
type CommandValidator struct {
	allowedExecutables map[string]bool
	blockedPatterns    []*regexp.Regexp
	maxLength          int
	maxArgs            int
}

// NewCommandValidator creates a new command validator
func NewCommandValidator() *CommandValidator {
	return &CommandValidator{
		allowedExecutables: map[string]bool{
			// Package managers
			"npm": true, "npx": true, "yarn": true, "pnpm": true,
			"pip": true, "pip3": true, "pipenv": true, "poetry": true,
			"cargo": true, "rustup": true,
			"go": true,
			"composer": true,
			"gem": true, "bundle": true,
			"nuget": true, "dotnet": true,
			"mvn": true, "gradle": true,

			// Build tools
			"make": true, "cmake": true, "ninja": true,
			"msbuild": true,

			// Runtimes
			"python": true, "python3": true,
			"node": true, "deno": true, "bun": true,
			"java": true, "javac": true,
			"ruby": true,
			"php": true,

			// Version control
			"git": true,

			// Containers
			"docker": true, "docker-compose": true, "podman": true,

			// Windows specific
			"powershell": true, "pwsh": true,
		},
		blockedPatterns: []*regexp.Regexp{
			// Shell operators
			regexp.MustCompile(`[;&|` + "`" + `$]`),
			// Redirections
			regexp.MustCompile(`[<>]`),
			// Command substitution
			regexp.MustCompile(`\$\([^)]*\)`),
			regexp.MustCompile("`[^`]*`"),
			// Environment variable expansion (potential info leak)
			regexp.MustCompile(`\$\{[^}]*\}`),
			regexp.MustCompile(`%[^%]+%`), // Windows env vars
			// Network commands (should be explicit)
			regexp.MustCompile(`(?i)^(curl|wget|nc|netcat|ssh|scp|rsync|ftp)\s`),
			// Dangerous commands
			regexp.MustCompile(`(?i)^(rm|del|rmdir|format|mkfs)\s`),
			regexp.MustCompile(`(?i)^(shutdown|reboot|init)\s`),
			// Script execution that bypasses validation
			regexp.MustCompile(`(?i)^(bash|sh|zsh|fish|csh|tcsh|ksh)\s+-c\s`),
			regexp.MustCompile(`(?i)^(cmd|powershell)\s+/c\s`),
		},
		maxLength: 4096,
		maxArgs:   100,
	}
}

// Validate validates a command string and returns a ValidatedCommand
func (v *CommandValidator) Validate(cmd string, workingDir string) (*ValidatedCommand, error) {
	// Trim whitespace
	cmd = strings.TrimSpace(cmd)

	// Check empty
	if cmd == "" {
		return nil, errors.NewUserError("CMD_EMPTY", "Command cannot be empty")
	}

	// Check length
	if len(cmd) > v.maxLength {
		return nil, errors.NewSecurityError("CMD_TOO_LONG",
			fmt.Sprintf("Command exceeds maximum length of %d", v.maxLength))
	}

	// Check for null bytes
	if strings.ContainsRune(cmd, '\x00') {
		return nil, errors.NewSecurityError("CMD_NULL_BYTE", "Command contains null bytes")
	}

	// Check for control characters
	for _, r := range cmd {
		if unicode.IsControl(r) && r != ' ' && r != '\t' {
			return nil, errors.NewSecurityError("CMD_CONTROL_CHAR",
				"Command contains control characters")
		}
	}

	// Check blocked patterns
	for _, pattern := range v.blockedPatterns {
		if pattern.MatchString(cmd) {
			return nil, errors.NewSecurityError("CMD_BLOCKED_PATTERN",
				fmt.Sprintf("Command matches blocked pattern: %s", pattern.String()))
		}
	}

	// Parse command into parts
	parts, err := parseCommand(cmd)
	if err != nil {
		return nil, errors.WrapWithCode(err, "CMD_PARSE_ERROR", "failed to parse command")
	}

	if len(parts) == 0 {
		return nil, errors.NewUserError("CMD_EMPTY", "Command is empty after parsing")
	}

	if len(parts) > v.maxArgs {
		return nil, errors.NewSecurityError("CMD_TOO_MANY_ARGS",
			fmt.Sprintf("Command has too many arguments (max %d)", v.maxArgs))
	}

	// Extract and validate executable
	executable := parts[0]
	executableBase := strings.ToLower(filepath.Base(executable))
	executableBase = strings.TrimSuffix(executableBase, ".exe")
	executableBase = strings.TrimSuffix(executableBase, ".cmd")
	executableBase = strings.TrimSuffix(executableBase, ".bat")

	if !v.allowedExecutables[executableBase] {
		return nil, errors.NewSecurityError("CMD_NOT_ALLOWED",
			fmt.Sprintf("Executable '%s' is not in allowlist", executable))
	}

	// Validate working directory if provided
	if workingDir != "" {
		if _, err := NewSafePath(workingDir, workingDir); err != nil {
			return nil, errors.WrapWithCode(err, "CMD_INVALID_WORKDIR", "invalid working directory")
		}
	}

	return &ValidatedCommand{
		executable: executable,
		args:       parts[1:],
		original:   cmd,
		workingDir: workingDir,
	}, nil
}

// parseCommand splits a command string respecting quotes
func parseCommand(cmd string) ([]string, error) {
	var parts []string
	var current strings.Builder
	var inQuote rune
	var escaped bool

	for _, r := range cmd {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' {
			escaped = true
			continue
		}

		if inQuote != 0 {
			if r == inQuote {
				inQuote = 0
			} else {
				current.WriteRune(r)
			}
			continue
		}

		switch r {
		case '"', '\'':
			inQuote = r
		case ' ', '\t':
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if inQuote != 0 {
		return nil, fmt.Errorf("unclosed quote")
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts, nil
}

// Executable returns the executable name
func (c *ValidatedCommand) Executable() string {
	return c.executable
}

// Args returns the command arguments
func (c *ValidatedCommand) Args() []string {
	return c.args
}

// String returns the original command string
func (c *ValidatedCommand) String() string {
	return c.original
}

// WorkingDir returns the working directory
func (c *ValidatedCommand) WorkingDir() string {
	return c.workingDir
}


