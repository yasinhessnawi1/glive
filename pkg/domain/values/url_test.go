package values_test

import (
	"testing"

	"github.com/glive/domain/values"
	"github.com/glive/testing/testutil"
)

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		provider string
		owner    string
		repo     string
	}{
		{"github https", "https://github.com/user/repo", false, "github", "user", "repo"},
		{"github https with .git", "https://github.com/user/repo.git", false, "github", "user", "repo"},
		{"github ssh", "git@github.com:user/repo.git", false, "github", "user", "repo"},
		{"github short form", "user/repo", false, "github", "user", "repo"},
		{"gitlab https", "https://gitlab.com/user/repo", false, "gitlab", "user", "repo"},
		{"bitbucket https", "https://bitbucket.org/user/repo", false, "bitbucket", "user", "repo"},
		{"empty url", "", true, "", "", ""},
		{"invalid url", "not-a-url", true, "", "", ""},
		{"localhost blocked", "http://localhost/repo", true, "", "", ""},
		{"file protocol blocked", "file:///etc/passwd", true, "", "", ""},
		{"path traversal blocked", "https://github.com/user/../etc", true, "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoURL, err := values.ParseRepoURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if repoURL.Provider() != tt.provider {
				t.Errorf("expected provider %q, got %q", tt.provider, repoURL.Provider())
			}
			if repoURL.Owner() != tt.owner {
				t.Errorf("expected owner %q, got %q", tt.owner, repoURL.Owner())
			}
			if repoURL.Repo() != tt.repo {
				t.Errorf("expected repo %q, got %q", tt.repo, repoURL.Repo())
			}
		})
	}
}

func TestNewURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid https", "https://github.com/user/repo", false},
		{"valid http", "http://github.com/user/repo", false},
		{"empty", "", true},
		{"localhost blocked", "http://localhost/repo", true},
		{"file protocol blocked", "file:///etc/passwd", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := values.NewURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if url.Value() == "" {
				t.Error("URL value should not be empty")
			}
		})
	}
}

func TestRepoURL_CloneURL(t *testing.T) {
	repoURL, err := values.ParseRepoURL("https://github.com/user/repo")
	testutil.AssertNoError(t, err)

	cloneURL := repoURL.CloneURL()
	expected := "https://github.com/user/repo.git"

	if cloneURL != expected {
		t.Errorf("expected clone URL %q, got %q", expected, cloneURL)
	}
}
