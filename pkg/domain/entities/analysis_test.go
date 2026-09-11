package entities_test

import (
	"testing"

	"github.com/glive/domain/entities"
	"github.com/glive/domain/values"
	"github.com/glive/testing/testutil"
)

func TestNewAnalysis(t *testing.T) {
	analysis := entities.NewAnalysis(entities.ProjectTypeNodeJS)

	if analysis.ProjectType() != entities.ProjectTypeNodeJS {
		t.Errorf("expected project type nodejs, got %s", analysis.ProjectType())
	}

	if analysis.IsSuspicious() {
		t.Error("analysis should not be suspicious by default")
	}

	if len(analysis.SuspiciousReasons()) != 0 {
		t.Error("analysis should have no suspicious reasons by default")
	}
}

func TestAnalysis_AddCommand(t *testing.T) {
	analysis := entities.NewAnalysis(entities.ProjectTypePython)

	workingDir, _ := values.NewPath("/tmp")
	cmd, err := entities.NewCommand("cmd-1", "Install", "pip install -r requirements.txt", workingDir, entities.StageSetup, true)
	testutil.AssertNoError(t, err)

	analysis.AddCommand(cmd)

	if len(analysis.Commands()) != 1 {
		t.Errorf("expected 1 command, got %d", len(analysis.Commands()))
	}

	if analysis.Commands()[0].Command() != "pip install -r requirements.txt" {
		t.Errorf("expected command %q, got %q", "pip install -r requirements.txt", analysis.Commands()[0].Command())
	}
}

func TestAnalysis_MarkSuspicious(t *testing.T) {
	analysis := entities.NewAnalysis(entities.ProjectTypeNodeJS)

	reasons := []string{"contains eval()", "suspicious pattern"}
	analysis.SetSuspicious(reasons)

	if !analysis.IsSuspicious() {
		t.Error("analysis should be marked as suspicious")
	}

	if len(analysis.SuspiciousReasons()) != len(reasons) {
		t.Errorf("expected %d suspicious reasons, got %d", len(reasons), len(analysis.SuspiciousReasons()))
	}
}
