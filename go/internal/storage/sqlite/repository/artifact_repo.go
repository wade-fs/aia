package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/artifact"
	"github.com/wade/aia/internal/storage/sqlite"
)

type ArtifactRepository struct {
	db *sqlite.DB
}

func NewArtifactRepository(db *sqlite.DB) *ArtifactRepository {
	return &ArtifactRepository{db: db}
}

func (r *ArtifactRepository) Create(ctx context.Context, a *artifact.Artifact) error {
	query := `
		INSERT INTO artifacts (id, task_id, execution_id, name, path, media_type, sha256, size, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}

	_, err := r.db.RawDB().ExecContext(
		ctx, query,
		a.ID, a.TaskID, a.ExecutionID, a.Name, a.Path, a.MediaType, a.SHA256, a.Size,
		a.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to create artifact: %w", err)
	}
	return nil
}

func (r *ArtifactRepository) ListByTask(ctx context.Context, taskID domain.ID) ([]artifact.Artifact, error) {
	query := `
		SELECT id, task_id, execution_id, name, path, media_type, sha256, size, created_at
		FROM artifacts WHERE task_id = ? ORDER BY created_at ASC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to list artifacts: %w", err)
	}
	defer rows.Close()

	var artifacts []artifact.Artifact
	for rows.Next() {
		var a artifact.Artifact
		var createdAt string
		if err := rows.Scan(&a.ID, &a.TaskID, &a.ExecutionID, &a.Name, &a.Path, &a.MediaType, &a.SHA256, &a.Size, &createdAt); err != nil {
			return nil, err
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		artifacts = append(artifacts, a)
	}
	return artifacts, nil
}
