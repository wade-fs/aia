package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/memory"
	"github.com/wade/aia/internal/storage/sqlite"
)

type MemoryQuery struct {
	Scope     memory.MemoryScope
	ProjectID domain.ID
	AgentID   domain.ID
	Type      memory.MemoryType
	Keyword   string
	Limit     int
}

type MemoryRepository struct {
	db *sqlite.DB
}

func NewMemoryRepository(db *sqlite.DB) *MemoryRepository {
	return &MemoryRepository{db: db}
}

func (r *MemoryRepository) Create(ctx context.Context, m *memory.Memory) error {
	query := `
		INSERT INTO memories (
			id, type, scope, project_id, agent_id, title, content,
			source_type, source_id, confidence, status, version,
			created_at, updated_at, last_used_at, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now

	var projID, agentID, srcID, lastUsed, expires sql.NullString
	if m.ProjectID != "" {
		projID = sql.NullString{String: m.ProjectID, Valid: true}
	}
	if m.AgentID != "" {
		agentID = sql.NullString{String: m.AgentID, Valid: true}
	}
	if m.SourceID != "" {
		srcID = sql.NullString{String: m.SourceID, Valid: true}
	}
	if m.LastUsedAt != nil {
		lastUsed = sql.NullString{String: m.LastUsedAt.Format(time.RFC3339), Valid: true}
	}
	if m.ExpiresAt != nil {
		expires = sql.NullString{String: m.ExpiresAt.Format(time.RFC3339), Valid: true}
	}

	_, err := r.db.RawDB().ExecContext(
		ctx, query,
		m.ID, string(m.Type), string(m.Scope), projID, agentID, m.Title, m.Content,
		m.SourceType, srcID, m.Confidence, m.Status, m.Version,
		m.CreatedAt.Format(time.RFC3339), m.UpdatedAt.Format(time.RFC3339), lastUsed, expires,
	)
	if err != nil {
		return fmt.Errorf("failed to create memory: %w", err)
	}
	return nil
}

func (r *MemoryRepository) Get(ctx context.Context, id domain.ID) (*memory.Memory, error) {
	query := `
		SELECT id, type, scope, project_id, agent_id, title, content,
		       source_type, source_id, confidence, status, version,
		       created_at, updated_at, last_used_at, expires_at
		FROM memories WHERE id = ?
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, id)

	var m memory.Memory
	var mType, scope, createdAt, updatedAt string
	var projID, agentID, srcID, lastUsed, expires sql.NullString

	err := row.Scan(
		&m.ID, &mType, &scope, &projID, &agentID, &m.Title, &m.Content,
		&m.SourceType, &srcID, &m.Confidence, &m.Status, &m.Version,
		&createdAt, &updatedAt, &lastUsed, &expires,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("memory not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get memory: %w", err)
	}

	m.Type = memory.MemoryType(mType)
	m.Scope = memory.MemoryScope(scope)
	if projID.Valid {
		m.ProjectID = projID.String
	}
	if agentID.Valid {
		m.AgentID = agentID.String
	}
	if srcID.Valid {
		m.SourceID = srcID.String
	}
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if lastUsed.Valid {
		lu, _ := time.Parse(time.RFC3339, lastUsed.String)
		m.LastUsedAt = &lu
	}
	if expires.Valid {
		ex, _ := time.Parse(time.RFC3339, expires.String)
		m.ExpiresAt = &ex
	}

	return &m, nil
}

func (r *MemoryRepository) Search(ctx context.Context, q MemoryQuery) ([]memory.Memory, error) {
	query := `
		SELECT id, type, scope, project_id, agent_id, title, content,
		       source_type, source_id, confidence, status, version,
		       created_at, updated_at, last_used_at, expires_at
		FROM memories WHERE status = 'ACTIVE'
	`
	var args []any
	if q.Scope != "" {
		query += " AND scope = ?"
		args = append(args, string(q.Scope))
	}
	if q.ProjectID != "" {
		query += " AND (project_id = ? OR scope = 'GLOBAL')"
		args = append(args, q.ProjectID)
	}
	if q.AgentID != "" {
		query += " AND (agent_id = ? OR scope = 'GLOBAL')"
		args = append(args, q.AgentID)
	}
	if q.Type != "" {
		query += " AND type = ?"
		args = append(args, string(q.Type))
	}
	if q.Keyword != "" {
		query += " AND (title LIKE ? OR content LIKE ?)"
		kw := "%" + strings.TrimSpace(q.Keyword) + "%"
		args = append(args, kw, kw)
	}
	query += " ORDER BY confidence DESC, created_at DESC"
	if q.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", q.Limit)
	}

	rows, err := r.db.RawDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}
	defer rows.Close()

	var memories []memory.Memory
	for rows.Next() {
		var m memory.Memory
		var mType, scope, createdAt, updatedAt string
		var projID, agentID, srcID, lastUsed, expires sql.NullString

		if err := rows.Scan(
			&m.ID, &mType, &scope, &projID, &agentID, &m.Title, &m.Content,
			&m.SourceType, &srcID, &m.Confidence, &m.Status, &m.Version,
			&createdAt, &updatedAt, &lastUsed, &expires,
		); err != nil {
			return nil, err
		}

		m.Type = memory.MemoryType(mType)
		m.Scope = memory.MemoryScope(scope)
		if projID.Valid {
			m.ProjectID = projID.String
		}
		if agentID.Valid {
			m.AgentID = agentID.String
		}
		if srcID.Valid {
			m.SourceID = srcID.String
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		if lastUsed.Valid {
			lu, _ := time.Parse(time.RFC3339, lastUsed.String)
			m.LastUsedAt = &lu
		}
		if expires.Valid {
			ex, _ := time.Parse(time.RFC3339, expires.String)
			m.ExpiresAt = &ex
		}
		memories = append(memories, m)
	}

	return memories, nil
}

func (r *MemoryRepository) TouchLastUsed(ctx context.Context, id domain.ID) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.RawDB().ExecContext(ctx, "UPDATE memories SET last_used_at = ? WHERE id = ?", now, id)
	return err
}

func (r *MemoryRepository) Delete(ctx context.Context, id domain.ID) error {
	_, err := r.db.RawDB().ExecContext(ctx, "UPDATE memories SET status = 'DELETED', updated_at = ? WHERE id = ?", time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// Candidates
func (r *MemoryRepository) CreateCandidate(ctx context.Context, c *memory.MemoryCandidate) error {
	query := `
		INSERT INTO memory_candidates (
			id, type, scope, project_id, agent_id, title, content,
			source_type, source_id, confidence, review_status, created_at, reviewed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.ReviewStatus == "" {
		c.ReviewStatus = memory.ReviewPending
	}

	var projID, agentID, srcID, reviewedAt sql.NullString
	if c.ProjectID != "" {
		projID = sql.NullString{String: c.ProjectID, Valid: true}
	}
	if c.AgentID != "" {
		agentID = sql.NullString{String: c.AgentID, Valid: true}
	}
	if c.SourceID != "" {
		srcID = sql.NullString{String: c.SourceID, Valid: true}
	}
	if c.ReviewedAt != nil {
		reviewedAt = sql.NullString{String: c.ReviewedAt.Format(time.RFC3339), Valid: true}
	}

	_, err := r.db.RawDB().ExecContext(
		ctx, query,
		c.ID, string(c.Type), string(c.Scope), projID, agentID, c.Title, c.Content,
		c.SourceType, srcID, c.Confidence, string(c.ReviewStatus),
		c.CreatedAt.Format(time.RFC3339), reviewedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create memory candidate: %w", err)
	}
	return nil
}

func (r *MemoryRepository) ListCandidates(ctx context.Context, status memory.ReviewStatus) ([]memory.MemoryCandidate, error) {
	query := `
		SELECT id, type, scope, project_id, agent_id, title, content,
		       source_type, source_id, confidence, review_status, created_at, reviewed_at
		FROM memory_candidates WHERE 1=1
	`
	var args []any
	if status != "" {
		query += " AND review_status = ?"
		args = append(args, string(status))
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.db.RawDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list memory candidates: %w", err)
	}
	defer rows.Close()

	var candidates []memory.MemoryCandidate
	for rows.Next() {
		var c memory.MemoryCandidate
		var mType, scope, revStatus, createdAt string
		var projID, agentID, srcID, reviewedAt sql.NullString

		if err := rows.Scan(
			&c.ID, &mType, &scope, &projID, &agentID, &c.Title, &c.Content,
			&c.SourceType, &srcID, &c.Confidence, &revStatus, &createdAt, &reviewedAt,
		); err != nil {
			return nil, err
		}

		c.Type = memory.MemoryType(mType)
		c.Scope = memory.MemoryScope(scope)
		c.ReviewStatus = memory.ReviewStatus(revStatus)
		if projID.Valid {
			c.ProjectID = projID.String
		}
		if agentID.Valid {
			c.AgentID = agentID.String
		}
		if srcID.Valid {
			c.SourceID = srcID.String
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if reviewedAt.Valid {
			ra, _ := time.Parse(time.RFC3339, reviewedAt.String)
			c.ReviewedAt = &ra
		}
		candidates = append(candidates, c)
	}

	return candidates, nil
}

func (r *MemoryRepository) GetCandidate(ctx context.Context, id domain.ID) (*memory.MemoryCandidate, error) {
	query := `
		SELECT id, type, scope, project_id, agent_id, title, content,
		       source_type, source_id, confidence, review_status, created_at, reviewed_at
		FROM memory_candidates WHERE id = ?
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, id)

	var c memory.MemoryCandidate
	var mType, scope, revStatus, createdAt string
	var projID, agentID, srcID, reviewedAt sql.NullString

	err := row.Scan(
		&c.ID, &mType, &scope, &projID, &agentID, &c.Title, &c.Content,
		&c.SourceType, &srcID, &c.Confidence, &revStatus, &createdAt, &reviewedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("memory candidate not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get memory candidate: %w", err)
	}

	c.Type = memory.MemoryType(mType)
	c.Scope = memory.MemoryScope(scope)
	c.ReviewStatus = memory.ReviewStatus(revStatus)
	if projID.Valid {
		c.ProjectID = projID.String
	}
	if agentID.Valid {
		c.AgentID = agentID.String
	}
	if srcID.Valid {
		c.SourceID = srcID.String
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if reviewedAt.Valid {
		ra, _ := time.Parse(time.RFC3339, reviewedAt.String)
		c.ReviewedAt = &ra
	}

	return &c, nil
}

func (r *MemoryRepository) UpdateCandidateStatus(ctx context.Context, id domain.ID, status memory.ReviewStatus) error {
	now := time.Now().UTC().Format(time.RFC3339)
	query := `UPDATE memory_candidates SET review_status = ?, reviewed_at = ? WHERE id = ?`
	res, err := r.db.RawDB().ExecContext(ctx, query, string(status), now, id)
	if err != nil {
		return fmt.Errorf("failed to update candidate status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("candidate not found: %s", id)
	}
	return nil
}
