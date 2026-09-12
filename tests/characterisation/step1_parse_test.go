package characterisation

import (
	"testing"

	"github.com/glive/domain/values"
	"github.com/glive/infrastructure/git"
)

// Step 1 - parse. setup_project.go:105 calls values.ParseRepoURL on the raw
// input and then values.NewURL on the result to build the Project's URL, so both
// have to accept the same shapes. GL0 T0a fixed exactly that: ParseRepoURL
// normalises an scp-style SSH remote to ssh://, a form NewURL then rejected, so
// a valid SSH repository could never become a Project.
//
// These cases pin all three accepted forms and the round-trip between the two
// constructors, so the T6 collapse cannot silently drop one.

func TestStep1_Parse_AcceptedURLForms(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantProvider string
		wantOwner    string
		wantRepo     string
		wantString   string // what ParseRepoURL stores and hands to NewURL
	}{
		{
			name:         "https",
			input:        "https://github.com/octocat/hello-world",
			wantProvider: "github",
			wantOwner:    "octocat",
			wantRepo:     "hello-world",
			wantString:   "https://github.com/octocat/hello-world",
		},
		{
			name:         "scp-style ssh is normalised to an ssh:// URL",
			input:        "git@github.com:octocat/hello-world.git",
			wantProvider: "github",
			wantOwner:    "octocat",
			wantRepo:     "hello-world",
			wantString:   "ssh://git@github.com/octocat/hello-world.git",
		},
		{
			name:         "short form is expanded to a github https URL",
			input:        "octocat/hello-world",
			wantProvider: "github",
			wantOwner:    "octocat",
			wantRepo:     "hello-world",
			wantString:   "https://github.com/octocat/hello-world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoURL, err := values.ParseRepoURL(tt.input)
			if err != nil {
				t.Fatalf("ParseRepoURL(%q) = error %v, want it accepted", tt.input, err)
			}

			if got := repoURL.Provider(); got != tt.wantProvider {
				t.Errorf("Provider() = %q, want %q", got, tt.wantProvider)
			}
			if got := repoURL.Owner(); got != tt.wantOwner {
				t.Errorf("Owner() = %q, want %q", got, tt.wantOwner)
			}
			if got := repoURL.Repo(); got != tt.wantRepo {
				t.Errorf("Repo() = %q, want %q", got, tt.wantRepo)
			}
			if got := repoURL.String(); got != tt.wantString {
				t.Errorf("String() = %q, want %q", got, tt.wantString)
			}

			// The round trip Execute actually performs at setup_project.go:105.
			// Before T0a this failed for the ssh case.
			if _, err := values.NewURL(repoURL.String()); err != nil {
				t.Errorf("NewURL(%q) = error %v - ParseRepoURL produced a form NewURL rejects", repoURL.String(), err)
			}
		})
	}
}

// TestStep1_Parse_CloneURLIsAlwaysHTTPS pins that whatever form came in, the URL
// handed to `git clone` is the https one. Step 3 depends on this.
func TestStep1_Parse_CloneURLIsAlwaysHTTPS(t *testing.T) {
	for _, in := range []string{
		"https://github.com/octocat/hello-world",
		"git@github.com:octocat/hello-world.git",
		"octocat/hello-world",
	} {
		t.Run(in, func(t *testing.T) {
			repoURL, err := values.ParseRepoURL(in)
			if err != nil {
				t.Fatalf("ParseRepoURL(%q) = error %v", in, err)
			}
			const want = "https://github.com/octocat/hello-world.git"
			if got := repoURL.CloneURL(); got != want {
				t.Errorf("CloneURL() = %q, want %q", got, want)
			}
		})
	}
}

// TestStep1_Parse_ProviderHosts pins the provider -> host mapping T0a introduced.
// bitbucket is the case the previous provider+".com" construction got wrong.
func TestStep1_Parse_ProviderHosts(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://github.com/o/r", "https://github.com/o/r.git"},
		{"https://gitlab.com/o/r", "https://gitlab.com/o/r.git"},
		{"https://bitbucket.org/o/r", "https://bitbucket.org/o/r.git"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			repoURL, err := values.ParseRepoURL(tt.input)
			if err != nil {
				t.Fatalf("ParseRepoURL(%q) = error %v", tt.input, err)
			}
			if got := repoURL.CloneURL(); got != tt.want {
				t.Errorf("CloneURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestStep1_Parse_Rejections pins what step 1 refuses. These overlap
// tests/security deliberately: the security suite asserts the SSRF policy, this
// asserts that Execute's first step is where the policy is applied.
func TestStep1_Parse_Rejections(t *testing.T) {
	for _, in := range []string{
		"",                                // empty
		"not-a-url",                       // unparseable
		"file:///etc/passwd",              // blocked scheme
		"http://localhost/repo",           // loopback
		"http://127.0.0.1/repo",           // loopback
		"https://github.com/owner/../etc", // traversal
		"https://example.com/owner/repo",  // unknown provider
	} {
		t.Run(in, func(t *testing.T) {
			if _, err := values.ParseRepoURL(in); err == nil {
				t.Errorf("ParseRepoURL(%q) was accepted, want rejected", in)
			}
		})
	}
}

// TestStep1_Parse_GitClientAgrees pins that the git layer parses the same forms
// the domain layer does. Execute builds a git client from the same input at
// setup_project.go:193, so a disagreement between the two parsers would surface
// as a clone failure after a successful parse.
func TestStep1_Parse_GitClientAgrees(t *testing.T) {
	for _, in := range []string{
		"https://github.com/octocat/hello-world",
		"git@github.com:octocat/hello-world.git",
	} {
		t.Run(in, func(t *testing.T) {
			client, err := git.ParseGitHubURL(in)
			if err != nil {
				t.Fatalf("git.ParseGitHubURL(%q) = error %v, but values.ParseRepoURL accepts it", in, err)
			}
			if client.Owner != "octocat" {
				t.Errorf("Owner = %q, want %q", client.Owner, "octocat")
			}
			if client.Name != "hello-world" {
				t.Errorf("Name = %q, want %q", client.Name, "hello-world")
			}
		})
	}
}
