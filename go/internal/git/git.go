package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Client struct {
	repoPath string
}

func NewClient(repoPath string) *Client {
	absPath, err := filepath.Abs(repoPath)
	if err == nil {
		repoPath = absPath
	}
	return &Client{repoPath: repoPath}
}

func (c *Client) RepoPath() string {
	return c.repoPath
}

func (c *Client) runGit(args ...string) (string, string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = c.repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

func (c *Client) IsRepo() bool {
	gitDir := filepath.Join(c.repoPath, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return true
	}
	return false
}

func (c *Client) Init() error {
	_, stderr, err := c.runGit("init")
	if err != nil {
		return fmt.Errorf("git init failed: %s: %w", stderr, err)
	}
	// Configure user for git commits in test environments if not set
	_, _, _ = c.runGit("config", "user.name", "Agent Runtime")
	_, _, _ = c.runGit("config", "user.email", "agent@runtime.local")
	return nil
}

func (c *Client) AddAndCommit(message string) error {
	_, stderr, err := c.runGit("add", "-A")
	if err != nil {
		return fmt.Errorf("git add failed: %s: %w", stderr, err)
	}
	_, stderr, err = c.runGit("commit", "-m", message)
	if err != nil {
		return fmt.Errorf("git commit failed: %s: %w", stderr, err)
	}
	return nil
}

// GetCurrentCommit returns the commit SHA of HEAD
func (c *Client) GetCurrentCommit() (string, error) {
	stdout, stderr, err := c.runGit("rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD failed: %s: %w", stderr, err)
	}
	return stdout, nil
}

// GetDiff returns diff against the base commit (or working tree diff if baseCommit is empty)
func (c *Client) GetDiff(baseCommit string) (string, error) {
	var args []string
	if baseCommit != "" {
		args = []string{"diff", baseCommit}
	} else {
		args = []string{"diff", "HEAD"}
	}

	stdout, stderr, err := c.runGit(args...)
	if err != nil {
		return "", fmt.Errorf("git diff failed: %s: %w", stderr, err)
	}
	return stdout, nil
}

// HasUncommittedChanges checks if there are uncommitted modifications or untracked files
func (c *Client) HasUncommittedChanges() (bool, error) {
	stdout, stderr, err := c.runGit("status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("git status --porcelain failed: %s: %w", stderr, err)
	}
	return len(stdout) > 0, nil
}

// GetChangedFiles returns list of changed files relative to baseCommit
func (c *Client) GetChangedFiles(baseCommit string) ([]string, error) {
	fileMap := make(map[string]struct{})

	// 1. Files modified/staged/committed relative to baseCommit
	if baseCommit != "" {
		stdout, stderr, err := c.runGit("diff", "--name-only", baseCommit)
		if err == nil && len(stdout) > 0 {
			for _, line := range strings.Split(stdout, "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					fileMap[line] = struct{}{}
				}
			}
		} else if err != nil && stderr != "" {
			return nil, fmt.Errorf("git diff --name-only failed: %s: %w", stderr, err)
		}
	}

	// 2. Untracked and uncommitted files from porcelain
	statusOut, stderr, err := c.runGit("status", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("git status failed: %s: %w", stderr, err)
	}
	if len(statusOut) > 0 {
		for _, line := range strings.Split(statusOut, "\n") {
			line = strings.TrimSpace(line)
			if len(line) > 3 {
				// line format: "?? filename" or " M filename"
				filePath := strings.TrimSpace(line[2:])
				if filePath != "" {
					fileMap[filePath] = struct{}{}
				}
			}
		}
	}

	files := make([]string, 0, len(fileMap))
	for f := range fileMap {
		files = append(files, f)
	}
	return files, nil
}

// CreateWorktree creates a worktree branch
func (c *Client) CreateWorktree(path, branch string) error {
	var args []string
	if branch != "" {
		args = []string{"worktree", "add", "-B", branch, path, "HEAD"}
	} else {
		args = []string{"worktree", "add", "--detach", path, "HEAD"}
	}

	_, stderr, err := c.runGit(args...)
	if err != nil {
		return fmt.Errorf("git worktree add failed: %s: %w", stderr, err)
	}
	return nil
}

// RemoveWorktree removes a worktree
func (c *Client) RemoveWorktree(path string) error {
	_, stderr, err := c.runGit("worktree", "remove", "--force", path)
	if err != nil {
		return fmt.Errorf("git worktree remove failed: %s: %w", stderr, err)
	}
	// Prune dead worktrees
	_, _, _ = c.runGit("worktree", "prune")
	return nil
}
