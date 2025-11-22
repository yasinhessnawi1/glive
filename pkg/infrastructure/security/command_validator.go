package security

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// CommandValidator validates commands before execution to prevent injection attacks
type CommandValidator struct {
	allowedCommands  map[string]bool
	blockedPatterns  []*regexp.Regexp
	maxArgLength     int
	maxCommandLength int
}

// NewCommandValidator creates a new command validator with default security rules
func NewCommandValidator() *CommandValidator {
	return &CommandValidator{
		allowedCommands: map[string]bool{
			// Python
			"python": true, "python3": true, "pip": true, "pip3": true,
			"py": true, "uvicorn": true, "gunicorn": true, "flask": true, "django-admin": true,
			"pytest": true, "mypy": true, "black": true, "ruff": true, "isort": true,
			"poetry": true, "pipenv": true,
			// Python virtual environment activation scripts
			"activate": true, "activate.bat": true, "activate.ps1": true,
			"deactivate": true, "deactivate.bat": true,
			// Node.js
			"node": true, "npm": true, "npx": true, "yarn": true, "pnpm": true, "bun": true,
			"tsc": true, "tsx": true, "vite": true, "webpack": true, "esbuild": true,
			"eslint": true, "prettier": true, "jest": true, "vitest": true, "mocha": true,
			// Go
			"go": true,
			// Rust
			"cargo": true, "rustc": true, "rustup": true,
			// Git
			"git": true,
			// Build tools
			"make": true, "cmake": true, "ninja": true, "msbuild": true,
			// Docker
			"docker": true, "docker-compose": true, "podman": true,
			// Java
			"java": true, "javac": true, "mvn": true, "mvnw": true, "gradle": true, "gradlew": true,
			"mvnw.cmd": true, "gradlew.bat": true,
			// .NET
			"dotnet": true, "nuget": true,
			// Shell (restricted)
			"powershell": true, "pwsh": true, "cmd": true, "bash": true, "sh": true, "zsh": true,
			"source": true, // for sourcing scripts
			"copy": true, "move": true, "del": true, "rm": true, "rmdir": true, "mkdir": true,
			"chdir": true, "cd": true, "pwd": true, "ls": true, "dir": true, "tree": true,
			"cat": true, "type": true, "echo": true, "pause": true, "cls": true, "clear": true,
			"exit": true, "help": true, "man": true, "info": true, "which": true, "where": true,
			// Package managers
			"apt": true, "apt-get": true, "yum": true, "dnf": true, "brew": true,
			"choco": true, "winget": true, "scoop": true,
			// Common scripts/wrappers
			"setup": true, "install": true, "build": true, "start": true, "run": true, "test": true,
		},
		blockedPatterns: []*regexp.Regexp{
			// Only block truly dangerous patterns - allow normal command chaining
			// Variable expansion that could be exploited
			regexp.MustCompile(`\$\{[^}]*\}`), // ${var} with content
			regexp.MustCompile(`\$\([^)]+\)`), // $(cmd) with content
			// Dangerous network commands opening connections
			regexp.MustCompile(`(?i)(nc|netcat)\s+(-[a-z]+\s+)*\d`), // nc with port numbers
			regexp.MustCompile(`(?i)telnet\s+\S+\s+\d`),             // telnet host port
			// Dangerous file operations on system roots
			regexp.MustCompile(`(?i)(rm|del)\s+(-rf?\s+)?[/\\]$`),         // rm / or rm \
			regexp.MustCompile(`(?i)(rm|del)\s+(-rf?\s+)?[/\\](etc|usr|Windows|System32)`), // system dirs
			regexp.MustCompile(`(?i)format\s+[a-zA-Z]:`),
			regexp.MustCompile(`(?i)diskpart`),
			// Redirection to system paths
			regexp.MustCompile(`(?i)>\s*/etc/`),
			regexp.MustCompile(`(?i)>\s*C:\\Windows`),
			regexp.MustCompile(`(?i)>\s*C:\\System32`),
			// Encoded commands piped to shell interpreters
			regexp.MustCompile(`(?i)base64\s+-d.*\|\s*(sh|bash|powershell|cmd)`),
		},
		maxArgLength:     4096,
		maxCommandLength: 8192,
	}
}

// Validation errors
var (
	ErrCommandTooLong             = fmt.Errorf("command exceeds maximum length")
	ErrEmptyCommand               = fmt.Errorf("command is empty")
	ErrCommandNotAllowed          = fmt.Errorf("command not in allowlist")
	ErrDangerousCommand           = fmt.Errorf("command contains dangerous patterns")
	ErrNullByteInArgument         = fmt.Errorf("null byte found in argument")
	ErrControlCharacterInArgument = fmt.Errorf("control character found in argument")
	ErrPathTraversal              = fmt.Errorf("path traversal detected")
)

// Validate validates a command before execution
func (v *CommandValidator) Validate(cmd string) error {
	// 1. Check command length
	if len(cmd) > v.maxCommandLength {
		return fmt.Errorf("%w: %d bytes (max %d)", ErrCommandTooLong, len(cmd), v.maxCommandLength)
	}

	// 2. Trim whitespace
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ErrEmptyCommand
	}

	// 3. Check for blocked patterns first (faster rejection)
	for _, pattern := range v.blockedPatterns {
		if pattern.MatchString(cmd) {
			return fmt.Errorf("%w: matches blocked pattern %s", ErrDangerousCommand, pattern.String())
		}
	}

	// 4. Parse command into parts (handling quotes)
	parts, err := v.parseCommand(cmd)
	if err != nil {
		return fmt.Errorf("failed to parse command: %w", err)
	}

	if len(parts) == 0 {
		return ErrEmptyCommand
	}

	// 5. Check if base command is allowed
	baseCmd := filepath.Base(parts[0])
	baseCmd = strings.TrimSuffix(baseCmd, ".exe")
	baseCmd = strings.ToLower(baseCmd)

	if !v.allowedCommands[baseCmd] {
		return fmt.Errorf("%w: %s", ErrCommandNotAllowed, baseCmd)
	}

	// 6. Validate all arguments
	for i, arg := range parts[1:] {
		if err := v.validateArgument(arg, i); err != nil {
			return fmt.Errorf("argument %d: %w", i+1, err)
		}
	}

	return nil
}

// parseCommand parses a command string handling quotes and escapes
// Note: On Windows, backslashes are path separators, not escape characters
func (v *CommandValidator) parseCommand(cmd string) ([]string, error) {
	var parts []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escapeNext := false

	for i, r := range cmd {
		if escapeNext {
			current.WriteRune(r)
			escapeNext = false
			continue
		}

		switch r {
		case '\\':
			// On Windows, backslash is a path separator, not an escape character
			// Only treat as escape if followed by special characters and in double quotes
			if inSingleQuote || inDoubleQuote {
				current.WriteRune(r)
			} else {
				// Check if next char is a special character that needs escaping
				// Otherwise treat as literal (Windows path separator)
				if i+1 < len(cmd) {
					nextChar := cmd[i+1]
					if nextChar == '"' || nextChar == '\'' || nextChar == '\\' || nextChar == ' ' {
						escapeNext = true
					} else {
						current.WriteRune(r) // Literal backslash (path separator)
					}
				} else {
					current.WriteRune(r) // Trailing backslash
				}
			}
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			} else {
				current.WriteRune(r)
			}
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			} else {
				current.WriteRune(r)
			}
		case ' ', '\t':
			if !inSingleQuote && !inDoubleQuote {
				if current.Len() > 0 {
					parts = append(parts, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}

		// Safety check for unclosed quotes
		if i == len(cmd)-1 {
			if inSingleQuote || inDoubleQuote {
				return nil, fmt.Errorf("unclosed quote")
			}
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts, nil
}

// validateArgument validates a single command argument
func (v *CommandValidator) validateArgument(arg string, index int) error {
	// Check length
	if len(arg) > v.maxArgLength {
		return fmt.Errorf("argument too long: %d bytes (max %d)", len(arg), v.maxArgLength)
	}

	// Check for null bytes
	if strings.ContainsRune(arg, '\x00') {
		return ErrNullByteInArgument
	}

	// Check for control characters (allow tab, newline, carriage return)
	for _, r := range arg {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return fmt.Errorf("%w: found %q", ErrControlCharacterInArgument, r)
		}
	}

	// Check for path traversal in arguments
	if strings.Contains(arg, "..") {
		// Allow if it's part of a legitimate path like "../node_modules" but block absolute traversal
		if strings.HasPrefix(arg, "../") || strings.HasPrefix(arg, "..\\") {
			return fmt.Errorf("%w: %s", ErrPathTraversal, arg)
		}
	}

	return nil
}

// AddAllowedCommand adds a command to the allowlist (for configuration)
func (v *CommandValidator) AddAllowedCommand(cmd string) {
	v.allowedCommands[strings.ToLower(cmd)] = true
}

// RemoveAllowedCommand removes a command from the allowlist
func (v *CommandValidator) RemoveAllowedCommand(cmd string) {
	delete(v.allowedCommands, strings.ToLower(cmd))
}

// IsAllowed checks if a command is in the allowlist
func (v *CommandValidator) IsAllowed(cmd string) bool {
	baseCmd := filepath.Base(cmd)
	baseCmd = strings.TrimSuffix(baseCmd, ".exe")
	return v.allowedCommands[strings.ToLower(baseCmd)]
}
