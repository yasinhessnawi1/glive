package values

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/glive/domain/errors"
)

// URL represents a validated GitHub URL
type URL struct {
	value string
}

var (
	githubHTTPS    = regexp.MustCompile(`^https://github\.com/([a-zA-Z0-9_.-]+)/([a-zA-Z0-9_.-]+?)(?:\.git)?/?$`)
	githubSSH      = regexp.MustCompile(`^git@github\.com:([a-zA-Z0-9_.-]+)/([a-zA-Z0-9_.-]+?)(?:\.git)?$`)
	gitlabHTTPS    = regexp.MustCompile(`^https://gitlab\.com/([a-zA-Z0-9_.-]+)/([a-zA-Z0-9_.-]+?)(?:\.git)?/?$`)
	bitbucketHTTPS = regexp.MustCompile(`^https://bitbucket\.org/([a-zA-Z0-9_.-]+)/([a-zA-Z0-9_.-]+?)(?:\.git)?/?$`)

	// Blocked patterns
	blockedPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(localhost|127\.0\.0\.1|0\.0\.0\.0|::1)`),
		regexp.MustCompile(`(?i)file://`),
		regexp.MustCompile(`(?i)\.\.`),        // Path traversal
		regexp.MustCompile(`(?i)[\x00-\x1f]`), // Control characters
	}
)

// RepoURL represents a validated repository URL with provider information
type RepoURL struct {
	original string
	parsed   *url.URL
	owner    string
	repo     string
	provider string
}

// NewURL creates a new validated URL
func NewURL(value string) (*URL, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.ErrInvalidURL.WithContext("reason", "empty URL")
	}

	// Check for blocked patterns
	for _, pattern := range blockedPatterns {
		if pattern.MatchString(value) {
			return nil, errors.ErrInvalidURL.WithContext("reason", "blocked pattern detected")
		}
	}

	// Basic URL validation
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") && !strings.HasPrefix(value, "git@") {
		// Try short form: owner/repo
		if matches := regexp.MustCompile(`^([a-zA-Z0-9_.-]+)/([a-zA-Z0-9_.-]+)$`).FindStringSubmatch(value); matches != nil {
			value = "https://github.com/" + matches[1] + "/" + matches[2]
		} else {
			return nil, errors.ErrInvalidURL.WithContext("reason", "invalid URL format")
		}
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return nil, errors.ErrInvalidURL.WithCause(err)
	}

	return &URL{value: parsed.String()}, nil
}

// ParseRepoURL parses a repository URL and returns a RepoURL with provider information
func ParseRepoURL(input string) (*RepoURL, error) {
	input = strings.TrimSpace(input)

	if input == "" {
		return nil, errors.ErrInvalidURL.WithContext("reason", "empty URL")
	}

	// Check for blocked patterns
	for _, pattern := range blockedPatterns {
		if pattern.MatchString(input) {
			return nil, errors.ErrInvalidURL.WithContext("reason", "blocked pattern detected")
		}
	}

	// Try to match known providers
	if matches := githubHTTPS.FindStringSubmatch(input); matches != nil {
		return newRepoURL(input, "github", matches[1], matches[2])
	}

	if matches := githubSSH.FindStringSubmatch(input); matches != nil {
		// Normalize SCP-like SSH URL to ssh:// scheme for url.Parse compatibility
		normalized := fmt.Sprintf("ssh://git@github.com/%s/%s.git", matches[1], matches[2])
		return newRepoURL(normalized, "github", matches[1], matches[2])
	}

	if matches := gitlabHTTPS.FindStringSubmatch(input); matches != nil {
		return newRepoURL(input, "gitlab", matches[1], matches[2])
	}

	if matches := bitbucketHTTPS.FindStringSubmatch(input); matches != nil {
		return newRepoURL(input, "bitbucket", matches[1], matches[2])
	}

	// Try short form: owner/repo
	if matches := regexp.MustCompile(`^([a-zA-Z0-9_.-]+)/([a-zA-Z0-9_.-]+)$`).FindStringSubmatch(input); matches != nil {
		expanded := "https://github.com/" + matches[1] + "/" + matches[2]
		return newRepoURL(expanded, "github", matches[1], matches[2])
	}

	return nil, errors.ErrInvalidURL.WithContext("input", input)
}

func newRepoURL(original, provider, owner, repo string) (*RepoURL, error) {
	// Additional validation
	if len(owner) > 100 || len(repo) > 100 {
		return nil, errors.ErrInvalidURL.WithContext("reason", "owner or repo name too long")
	}

	// Parse full URL
	parsed, err := url.Parse(original)
	if err != nil {
		return nil, errors.ErrInvalidURL.WithCause(err)
	}

	return &RepoURL{
		original: original,
		parsed:   parsed,
		owner:    owner,
		repo:     repo,
		provider: provider,
	}, nil
}

// Value returns the string value of the URL
func (u *URL) Value() string {
	return u.value
}

// Equals checks if two URLs are equal
func (u *URL) Equals(other *URL) bool {
	if other == nil {
		return false
	}
	return u.value == other.value
}

// ParseOwnerAndName parses owner and repository name from GitHub URL
func (u *URL) ParseOwnerAndName() (owner string, name string, err error) {
	parsed, err := url.Parse(u.value)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse URL: %w", err)
	}

	path := strings.Trim(parsed.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid GitHub URL format")
	}

	return parts[0], strings.TrimSuffix(parts[1], ".git"), nil
}

// String returns the string representation
func (u *URL) String() string {
	return u.value
}

// Normalize returns a normalized version of the URL for comparison
func (u *URL) Normalize() string {
	normalized := strings.ToLower(u.value)
	normalized = strings.TrimSuffix(normalized, ".git")
	normalized = strings.TrimSuffix(normalized, "/")
	return normalized
}

// RepoURL methods

// String returns the original URL string
func (r *RepoURL) String() string {
	return r.original
}

// CloneURL returns the URL for cloning
func (r *RepoURL) CloneURL() string {
	return fmt.Sprintf("https://%s.com/%s/%s.git", r.provider, r.owner, r.repo)
}

// Owner returns the repository owner
func (r *RepoURL) Owner() string {
	return r.owner
}

// Repo returns the repository name
func (r *RepoURL) Repo() string {
	return r.repo
}

// Provider returns the Git provider (github, gitlab, bitbucket)
func (r *RepoURL) Provider() string {
	return r.provider
}

// Parsed returns the parsed URL
func (r *RepoURL) Parsed() *url.URL {
	return r.parsed
}
