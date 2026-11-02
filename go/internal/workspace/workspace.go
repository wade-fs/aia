package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/artifact"
)

type Manager struct {
	baseDir string
}

func NewManager(baseDir string) (*Manager, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base workspace directory %s: %w", baseDir, err)
	}
	return &Manager{baseDir: baseDir}, nil
}

func (m *Manager) PrepareWorkspace(executionID domain.ID) (string, error) {
	wsPath := filepath.Join(m.baseDir, string(executionID))
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create execution workspace: %w", err)
	}
	return wsPath, nil
}

func (m *Manager) IsPathContained(wsPath, targetPath string) bool {
	absWS, err1 := filepath.Abs(wsPath)
	absTarget, err2 := filepath.Abs(targetPath)
	if err1 != nil || err2 != nil {
		return false
	}
	return strings.HasPrefix(absTarget, absWS)
}

func (m *Manager) CollectArtifacts(taskID, executionID domain.ID, wsPath string) ([]artifact.Artifact, error) {
	var artifacts []artifact.Artifact

	err := filepath.Walk(wsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(wsPath, path)
		if strings.HasPrefix(relPath, ".") {
			return nil
		}

		hash, size, err := computeSHA256(path)
		if err != nil {
			return nil
		}

		mediaType := detectMediaType(path)
		art := artifact.Artifact{
			ID:          uuid.New().String(),
			TaskID:      taskID,
			ExecutionID: executionID,
			Name:        info.Name(),
			Path:        path,
			MediaType:   mediaType,
			SHA256:      hash,
			Size:        size,
			CreatedAt:   time.Now().UTC(),
		}
		artifacts = append(artifacts, art)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to collect artifacts: %w", err)
	}
	return artifacts, nil
}

func computeSHA256(filePath string) (string, int64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	h := sha256.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}

	return hex.EncodeToString(h.Sum(nil)), size, nil
}

func detectMediaType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "text/x-go"
	case ".md":
		return "text/markdown"
	case ".txt":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".yaml", ".yml":
		return "application/x-yaml"
	case ".sh":
		return "application/x-sh"
	case ".html":
		return "text/html"
	default:
		return "application/octet-stream"
	}
}

// FileChange records a filesystem modification within a workspace (Plan.md §4.2).
type FileChange struct {
	RelPath    string `json:"rel_path"`
	ChangeType string `json:"change_type"` // ADDED, MODIFIED, DELETED
	SHA256     string `json:"sha256,omitempty"`
}

// ResolveWorkspaceRoot resolves ~, relative paths, and ensures the target directory exists.
func ResolveWorkspaceRoot(rawPath string) (string, error) {
	if rawPath == "" {
		return os.Getwd()
	}
	if strings.HasPrefix(rawPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home dir: %w", err)
		}
		if rawPath == "~" {
			rawPath = home
		} else if strings.HasPrefix(rawPath, "~/") {
			rawPath = filepath.Join(home, rawPath[2:])
		}
	}
	abs, err := filepath.Abs(rawPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path %s: %w", rawPath, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("workspace directory does not exist: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("workspace path is not a directory: %s", abs)
	}
	return abs, nil
}

// ValidateBoundaries verifies that targetPath resides strictly within wsRoot (anti-path-traversal).
func ValidateBoundaries(wsRoot, targetPath string) (string, error) {
	absRoot, err := filepath.Abs(wsRoot)
	if err != nil {
		return "", fmt.Errorf("invalid workspace root %s: %w", wsRoot, err)
	}
	var absTarget string
	if filepath.IsAbs(targetPath) {
		absTarget = filepath.Clean(targetPath)
	} else {
		absTarget = filepath.Clean(filepath.Join(absRoot, targetPath))
	}

	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
		return "", fmt.Errorf("access denied: path %s escapes workspace root %s", targetPath, wsRoot)
	}
	return absTarget, nil
}

// TrackWorkspaceChanges compares current file hashes against previousHashes and returns the updated map and delta.
func TrackWorkspaceChanges(wsRoot string, previousHashes map[string]string) (map[string]string, []FileChange, error) {
	currentHashes := make(map[string]string)
	var changes []FileChange

	err := filepath.Walk(wsRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(wsRoot, path)
		if rel == "." || strings.HasPrefix(rel, ".git") || strings.HasPrefix(rel, ".cache") {
			if info.IsDir() && (rel == ".git" || rel == ".cache") {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			return nil
		}

		hash, _, err := computeSHA256(path)
		if err != nil {
			return nil
		}
		currentHashes[rel] = hash

		prev, exists := previousHashes[rel]
		if !exists {
			changes = append(changes, FileChange{
				RelPath:    rel,
				ChangeType: "ADDED",
				SHA256:     hash,
			})
		} else if prev != hash {
			changes = append(changes, FileChange{
				RelPath:    rel,
				ChangeType: "MODIFIED",
				SHA256:     hash,
			})
		}
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to scan workspace %s: %w", wsRoot, err)
	}

	for prevRel := range previousHashes {
		if _, exists := currentHashes[prevRel]; !exists {
			changes = append(changes, FileChange{
				RelPath:    prevRel,
				ChangeType: "DELETED",
			})
		}
	}

	return currentHashes, changes, nil
}

// CreateIsolatedWorktree creates a dedicated Git worktree branch for safe modifications (Plan.md §4.4, §10).
func CreateIsolatedWorktree(repoRoot, branchName, worktreeDir string) error {
	if _, err := os.Stat(filepath.Join(repoRoot, ".git")); err != nil {
		return fmt.Errorf("repo root %s is not a git repository: %w", repoRoot, err)
	}
	_ = os.RemoveAll(worktreeDir)

	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreeDir, "HEAD")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		// If branch already exists, try checking it out directly
		cmdReuse := exec.Command("git", "worktree", "add", worktreeDir, branchName)
		cmdReuse.Dir = repoRoot
		if out2, err2 := cmdReuse.CombinedOutput(); err2 != nil {
			return fmt.Errorf("git worktree add failed: %s (%s): %w", string(out), string(out2), err)
		}
	}
	return nil
}

// RemoveIsolatedWorktree detaches and cleans up an isolated worktree.
func RemoveIsolatedWorktree(repoRoot, worktreeDir string) error {
	cmd := exec.Command("git", "worktree", "remove", "--force", worktreeDir)
	cmd.Dir = repoRoot
	_ = cmd.Run()
	_ = os.RemoveAll(worktreeDir)
	return nil
}

// GetWorktreeDiff inspects uncommitted diffs inside a worktree.
func GetWorktreeDiff(worktreeDir string) (string, error) {
	cmd := exec.Command("git", "diff", "HEAD")
	cmd.Dir = worktreeDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git diff failed: %s: %w", string(out), err)
	}
	return string(out), nil
}
