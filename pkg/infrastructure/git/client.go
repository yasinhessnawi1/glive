package git

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
	ErrInvalidURL        = errors.New("invalid GitHub URL")
	ErrCloneFailed       = errors.New("failed to clone repository")
	ErrDirectoryConflict = errors.New("directory conflict")
)

// ConflictError represents a directory conflict that needs user resolution
type ConflictError struct {
	Path    string
	IsGitRepo bool
	Message string
}

func (e *ConflictError) Error() string {
	return e.Message
}

func (e *ConflictError) Is(target error) bool {
	return target == ErrDirectoryConflict
}

// Client handles GitHub repository operations
type Client struct {
	URL       string
	Owner     string
	Name      string
	LocalPath string
}

// ParseGitHubURL parses and validates a GitHub URL
func ParseGitHubURL(gitURL string) (*Client, error) {
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

	return &Client{
		URL:   gitURL,
		Owner: owner,
		Name:  name,
	}, nil
}

// Clone clones the repository to the specified directory using git command
func (c *Client) Clone(workspaceDir string, progress io.Writer) error {
	// Create workspace directory if it doesn't exist
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Generate local path
	c.LocalPath = filepath.Join(workspaceDir, c.Name)

	// Check if directory already exists
	if _, err := os.Stat(c.LocalPath); err == nil {
		// Check if it's a valid git repository
		gitDir := filepath.Join(c.LocalPath, ".git")
		if info, gitErr := os.Stat(gitDir); gitErr == nil && info.IsDir() {
			// It's already a git repository, use it
			if progress != nil {
				fmt.Fprintf(progress, "   ℹ️  Repository already exists, using existing clone...\n")
			}
			return nil
		}

		// Directory exists but is not a git repo - return conflict error for user resolution
		return &ConflictError{
			Path:      c.LocalPath,
			IsGitRepo: false,
			Message:   fmt.Sprintf("directory already exists but is not a git repository: %s", c.LocalPath),
		}
	}

	// Check if git is installed
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not in PATH: %w", err)
	}

	// Clone using git command (shallow clone for speed)
	cmd := exec.Command("git", "clone", "--depth", "1", c.URL, c.LocalPath)
	cmd.Stdout = progress
	cmd.Stderr = progress

	if err := cmd.Run(); err != nil {
		// Clean up failed clone
		os.RemoveAll(c.LocalPath)
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

// CloneWithAuth clones a private repository with authentication
func (c *Client) CloneWithAuth(workspaceDir string, username, token string, progress io.Writer) error {
	// Create workspace directory if it doesn't exist
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Generate local path
	c.LocalPath = filepath.Join(workspaceDir, c.Name)

	// Check if directory already exists
	if _, err := os.Stat(c.LocalPath); err == nil {
		return fmt.Errorf("directory already exists: %s", c.LocalPath)
	}

	// Check if git is installed
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not in PATH: %w", err)
	}

	// Build authenticated URL (https://username:token@github.com/owner/repo.git)
	authURL := strings.Replace(c.URL, "https://", fmt.Sprintf("https://%s:%s@", username, token), 1)

	// Clone using git command with auth
	cmd := exec.Command("git", "clone", "--depth", "1", authURL, c.LocalPath)
	cmd.Stdout = progress
	cmd.Stderr = progress

	if err := cmd.Run(); err != nil {
		// Clean up failed clone
		os.RemoveAll(c.LocalPath)
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

// GetLocalPath returns the local filesystem path
func (c *Client) GetLocalPath() string {
	return c.LocalPath
}

// IsCloned checks if the repository is already cloned
func (c *Client) IsCloned() bool {
	if c.LocalPath == "" {
		return false
	}

	gitDir := filepath.Join(c.LocalPath, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}

// DeleteExisting removes the existing directory at LocalPath
func (c *Client) DeleteExisting() error {
	if c.LocalPath == "" {
		return fmt.Errorf("no local path set")
	}
	return os.RemoveAll(c.LocalPath)
}

// RenameExisting renames the existing directory with a backup suffix
func (c *Client) RenameExisting() (string, error) {
	if c.LocalPath == "" {
		return "", fmt.Errorf("no local path set")
	}

	// Find a unique backup name
	backupPath := c.LocalPath + "_backup"
	counter := 1
	for {
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			break
		}
		backupPath = fmt.Sprintf("%s_backup_%d", c.LocalPath, counter)
		counter++
		if counter > 100 {
			return "", fmt.Errorf("too many backup directories exist")
		}
	}

	if err := os.Rename(c.LocalPath, backupPath); err != nil {
		return "", fmt.Errorf("failed to rename directory: %w", err)
	}
	return backupPath, nil
}

