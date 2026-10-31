package workspace

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/wade/aia/agent-runtime/internal/git"
)

var (
	ErrWorkspaceNotFound = errors.New("workspace not found")
)

type Workspace struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	BaseCommit string `json:"base_commit"`
	TaskID     string `json:"task_id"`
}

type Manager struct {
	baseDir    string
	sourceRepo string
	gitClient  *git.Client
}

func NewManager(baseDir, sourceRepo string) *Manager {
	if absBase, err := filepath.Abs(baseDir); err == nil {
		baseDir = absBase
	}
	if sourceRepo != "" {
		if absSource, err := filepath.Abs(sourceRepo); err == nil {
			sourceRepo = absSource
		}
	}

	var client *git.Client
	if sourceRepo != "" {
		client = git.NewClient(sourceRepo)
	}
	return &Manager{
		baseDir:    baseDir,
		sourceRepo: sourceRepo,
		gitClient:  client,
	}
}

func (m *Manager) BaseDir() string {
	return m.baseDir
}

func (m *Manager) Exists(taskID string) bool {
	wsPath := filepath.Join(m.baseDir, taskID)
	info, err := os.Stat(wsPath)
	return err == nil && info.IsDir()
}

func (m *Manager) Get(taskID string) (*Workspace, error) {
	wsPath := filepath.Join(m.baseDir, taskID)
	if !m.Exists(taskID) {
		return nil, ErrWorkspaceNotFound
	}
	client := git.NewClient(wsPath)
	baseCommit, _ := client.GetCurrentCommit()
	return &Workspace{
		ID:         taskID,
		Path:       wsPath,
		BaseCommit: baseCommit,
		TaskID:     taskID,
	}, nil
}

func (m *Manager) Create(taskID string) (*Workspace, error) {
	if err := os.MkdirAll(m.baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base workspaces directory: %w", err)
	}

	wsPath := filepath.Join(m.baseDir, taskID)
	if m.Exists(taskID) {
		return m.Get(taskID)
	}

	// Strategy:
	// If sourceRepo exists and is an independent git repository with at least one commit, attempt git worktree add.
	// Otherwise, copy source project files into wsPath and git init an isolated workspace repository.
	createdViaWorktree := false
	if m.sourceRepo != "" && m.gitClient != nil && m.gitClient.IsRepo() {
		if _, err := m.gitClient.GetCurrentCommit(); err == nil {
			branchName := fmt.Sprintf("task-%s", taskID)
			if err := m.gitClient.CreateWorktree(wsPath, branchName); err == nil {
				createdViaWorktree = true
			}
		}
	}

	if !createdViaWorktree {
		if err := os.MkdirAll(wsPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create workspace dir: %w", err)
		}

		if m.sourceRepo != "" {
			if _, err := os.Stat(m.sourceRepo); err == nil {
				if err := copyDir(m.sourceRepo, wsPath); err != nil {
					return nil, fmt.Errorf("failed to copy source project to workspace: %w", err)
				}
			}
		}

		localGit := git.NewClient(wsPath)
		if !localGit.IsRepo() {
			if err := localGit.Init(); err != nil {
				return nil, fmt.Errorf("failed to init git in workspace: %w", err)
			}
			if err := localGit.AddAndCommit("Initial commit for workspace"); err != nil {
				return nil, fmt.Errorf("failed to make initial commit in workspace: %w", err)
			}
		}
	}

	localGit := git.NewClient(wsPath)
	baseCommit, err := localGit.GetCurrentCommit()
	if err != nil {
		return nil, fmt.Errorf("failed to get base commit for workspace: %w", err)
	}

	return &Workspace{
		ID:         uuid.New().String(),
		Path:       wsPath,
		BaseCommit: baseCommit,
		TaskID:     taskID,
	}, nil
}

func (m *Manager) Cleanup(ws *Workspace) error {
	if ws == nil || ws.Path == "" {
		return nil
	}
	if m.gitClient != nil && m.gitClient.IsRepo() {
		_ = m.gitClient.RemoveWorktree(ws.Path)
	}
	return os.RemoveAll(ws.Path)
}

func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		// Skip existing .git directory when copying
		if entry.Name() == ".git" {
			continue
		}

		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				return err
			}
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
