package entities_test

import (
	"testing"

	"github.com/glive/domain/entities"
	"github.com/glive/domain/values"
	"github.com/glive/testing/testutil"
)

func TestNewCommand(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		description string
		workingDir  string
		wantErr     bool
	}{
		{"valid command", "npm install", "Install dependencies", "/tmp", false},
		{"empty command", "", "Empty", "/tmp", true},
		{"empty working dir", "npm install", "Install", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var workingDir *values.Path
			var err error
			if tt.workingDir != "" {
				workingDir, err = values.NewPath(tt.workingDir)
				if err != nil {
					t.Fatalf("failed to create path: %v", err)
				}
			}

			cmd, err := entities.NewCommand("cmd-1", tt.description, tt.command, workingDir, entities.StageSetup, true)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cmd.Command() != tt.command {
				t.Errorf("expected command %q, got %q", tt.command, cmd.Command())
			}
			if cmd.Description() != tt.description {
				t.Errorf("expected description %q, got %q", tt.description, cmd.Description())
			}
		})
	}
}

func TestCommand_Required(t *testing.T) {
	workingDir, _ := values.NewPath("/tmp")
	cmd, err := entities.NewCommand("cmd-1", "Install", "npm install", workingDir, entities.StageSetup, true)
	testutil.AssertNoError(t, err)

	if !cmd.Required() {
		t.Error("command should be required")
	}

	cmd2, err := entities.NewCommand("cmd-2", "Optional", "npm test", workingDir, entities.StageSetup, false)
	testutil.AssertNoError(t, err)

	if cmd2.Required() {
		t.Error("command should not be required")
	}
}
