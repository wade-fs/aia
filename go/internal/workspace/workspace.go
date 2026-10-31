package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
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
