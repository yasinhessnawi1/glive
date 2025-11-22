package repo

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	ErrInvalidURL = errors.New("invalid GitHub URL")
	ErrCloneFailed = errors.New("failed to clone repository")
)

// Repository handles GitHub repository operations
type Repository struct {
	URL       string
	Owner     string
	Name      string
	LocalPath string
}

// ParseGitHubURL parses and validates a GitHub URL
func ParseGitHubURL(gitURL string) (*Repository, error) {
	// Clean the URL
	gitURL = strings.TrimSpace(gitURL)

	// Support various GitHub URL formats
	// - https://github.com/owner/repo
	// - https://github.com/owner/repo.git
	// - git@github.com:owner/repo.git
	// - owner/repo

	var owner, name string

	// Handle SSH format
	if strings.HasPrefix(gitURL, "git@github.com:") {
		parts := strings.TrimPrefix(gitURL, "git@github.com:")
		parts = strings.TrimSuffix(parts, ".git")
		segments := strings.Split(parts, "/")
		if len(segments) != 2 {
			return nil, ErrInvalidURL
		}
		owner, name = segments[0], segments[1]
	} else if strings.Contains(gitURL, "github.com") {
		// Handle HTTPS format
		u, err := url.Parse(gitURL)
		if err != nil {
			return nil, ErrInvalidURL
		}

		path := strings.Trim(u.Path, "/")
		path = strings.TrimSuffix(path, ".git")
		segments := strings.Split(path, "/")

		if len(segments) < 2 {
			return nil, ErrInvalidURL
		}
		owner, name = segments[0], segments[1]
	} else if strings.Count(gitURL, "/") == 1 && !strings.Contains(gitURL, " ") {
		// Handle shorthand format: owner/repo
		segments := strings.Split(gitURL, "/")
		owner, name = segments[0], segments[1]
		gitURL = fmt.Sprintf("https://github.com/%s/%s", owner, name)
	} else {
		return nil, ErrInvalidURL
	}

	if owner == "" || name == "" {
		return nil, ErrInvalidURL
	}

	return &Repository{
		URL:   gitURL,
		Owner: owner,
		Name:  name,
	}, nil
}

// Clone clones the repository to the specified directory using git command
func (r *Repository) Clone(workspaceDir string, progress io.Writer) error {
	// Create workspace directory if it doesn't exist
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Generate local path
	r.LocalPath = filepath.Join(workspaceDir, r.Name)

	// Check if directory already exists
	if _, err := os.Stat(r.LocalPath); err == nil {
		// Check if it's a valid git repository
		gitDir := filepath.Join(r.LocalPath, ".git")
		if info, gitErr := os.Stat(gitDir); gitErr == nil && info.IsDir() {
			// It's already a git repository, use it
			if progress != nil {
				fmt.Fprintf(progress, "   ℹ️  Repository already exists, using existing clone...\n")
			}
			return nil
		}

		// Directory exists but is not a git repo - error
		return fmt.Errorf("directory already exists but is not a git repository: %s\nPlease remove it or use a different workspace directory", r.LocalPath)
	}

	// Check if git is installed
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not in PATH: %w", err)
	}

	// Clone using git command (shallow clone for speed)
	cmd := exec.Command("git", "clone", "--depth", "1", r.URL, r.LocalPath)
	cmd.Stdout = progress
	cmd.Stderr = progress

	if err := cmd.Run(); err != nil {
		// Clean up failed clone
		os.RemoveAll(r.LocalPath)
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

// CloneWithAuth clones a private repository with authentication
func (r *Repository) CloneWithAuth(workspaceDir string, username, token string, progress io.Writer) error {
	// Create workspace directory if it doesn't exist
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Generate local path
	r.LocalPath = filepath.Join(workspaceDir, r.Name)

	// Check if directory already exists
	if _, err := os.Stat(r.LocalPath); err == nil {
		return fmt.Errorf("directory already exists: %s", r.LocalPath)
	}

	// Check if git is installed
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not in PATH: %w", err)
	}

	// Build authenticated URL (https://username:token@github.com/owner/repo.git)
	authURL := strings.Replace(r.URL, "https://", fmt.Sprintf("https://%s:%s@", username, token), 1)

	// Clone using git command with auth
	cmd := exec.Command("git", "clone", "--depth", "1", authURL, r.LocalPath)
	cmd.Stdout = progress
	cmd.Stderr = progress

	if err := cmd.Run(); err != nil {
		// Clean up failed clone
		os.RemoveAll(r.LocalPath)
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

// GetLocalPath returns the local filesystem path
func (r *Repository) GetLocalPath() string {
	return r.LocalPath
}

// IsCloned checks if the repository is already cloned
func (r *Repository) IsCloned() bool {
	if r.LocalPath == "" {
		return false
	}

	gitDir := filepath.Join(r.LocalPath, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}
