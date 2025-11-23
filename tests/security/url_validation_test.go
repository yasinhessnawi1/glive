package security

import (
	"testing"

	"github.com/glive/domain/values"
)

func TestURLValidation_Security(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid github", "https://github.com/user/repo", false},
		{"localhost blocked", "http://localhost/repo", true},
		{"127.0.0.1 blocked", "http://127.0.0.1/repo", true},
		{"0.0.0.0 blocked", "http://0.0.0.0/repo", true},
		{"file protocol blocked", "file:///etc/passwd", true},
		{"path traversal blocked", "https://github.com/user/../etc", true},
		{"control characters blocked", "https://github.com/user/repo\x00", true},
		{"internal ip blocked", "http://192.168.1.1/repo", true},
		{"ssrf attempt blocked", "http://169.254.169.254/latest/meta-data", true},
		{"private ip blocked", "http://10.0.0.1/repo", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := values.ParseRepoURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected malicious URL to be blocked: %s", tt.url)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for valid URL: %v", err)
				}
			}
		})
	}
}

func TestURLValidation_ProviderSupport(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		provider string
	}{
		{"github https", "https://github.com/user/repo", "github"},
		{"github ssh", "git@github.com:user/repo.git", "github"},
		{"gitlab https", "https://gitlab.com/user/repo", "gitlab"},
		{"bitbucket https", "https://bitbucket.org/user/repo", "bitbucket"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoURL, err := values.ParseRepoURL(tt.url)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if repoURL.Provider() != tt.provider {
				t.Errorf("expected provider %q, got %q", tt.provider, repoURL.Provider())
			}
		})
	}
}

func TestURLValidation_ShortForm(t *testing.T) {
	repoURL, err := values.ParseRepoURL("user/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repoURL.Provider() != "github" {
		t.Errorf("expected provider github for short form, got %q", repoURL.Provider())
	}

	if repoURL.Owner() != "user" {
		t.Errorf("expected owner user, got %q", repoURL.Owner())
	}

	if repoURL.Repo() != "repo" {
		t.Errorf("expected repo repo, got %q", repoURL.Repo())
	}
}
