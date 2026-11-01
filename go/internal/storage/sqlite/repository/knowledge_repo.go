package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wade/aia/internal/domain/knowledge"
	"github.com/wade/aia/internal/storage/sqlite"
)

type KnowledgeRepository struct {
	db *sqlite.DB
}

func NewKnowledgeRepository(db *sqlite.DB) *KnowledgeRepository {
	return &KnowledgeRepository{db: db}
}

// CreateSource stores a source with its 5-dimensional scores.
func (r *KnowledgeRepository) CreateSource(ctx context.Context, s *knowledge.Source) error {
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}
	if s.MetadataJSON == "" {
		s.MetadataJSON = "{}"
	}

	query := `
	INSERT INTO sources (
		id, source_type, uri, publisher,
		authority_score, directness_score, timeliness_score, independence_score, reproducibility_score,
		metadata_json, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	createdStr := s.CreatedAt.UTC().Format(time.RFC3339)

	_, err := r.db.RawDB().ExecContext(ctx, query,
		s.ID, s.SourceType, s.URI, s.Publisher,
		s.AuthorityScore, s.DirectnessScore, s.TimelinessScore, s.IndependenceScore, s.ReproducibilityScore,
		s.MetadataJSON, createdStr,
	)
	return err
}

// GetSource retrieves a source by ID.
func (r *KnowledgeRepository) GetSource(ctx context.Context, id string) (*knowledge.Source, error) {
	query := `
	SELECT id, source_type, uri, publisher,
	       authority_score, directness_score, timeliness_score, independence_score, reproducibility_score,
	       metadata_json, created_at
	FROM sources WHERE id = ?`

	s := &knowledge.Source{}
	var createdStr string

	err := r.db.RawDB().QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.SourceType, &s.URI, &s.Publisher,
		&s.AuthorityScore, &s.DirectnessScore, &s.TimelinessScore, &s.IndependenceScore, &s.ReproducibilityScore,
		&s.MetadataJSON, &createdStr,
	)
	if err != nil {
		return nil, err
	}
	s.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	return s, nil
}

// CreateEvidence stores a piece of evidence grounded in a source.
func (r *KnowledgeRepository) CreateEvidence(ctx context.Context, e *knowledge.Evidence) error {
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	if e.CapturedAt.IsZero() {
		e.CapturedAt = time.Now().UTC()
	}
	if e.DataVersion == "" {
		e.DataVersion = "1.0"
	}

	query := `
	INSERT INTO evidence (
		id, source_id, content_snippet, content_hash, capture_method, data_version, published_at, captured_at, valid_until, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var pubStr, validStr *string
	if e.PublishedAt != nil {
		s := e.PublishedAt.UTC().Format(time.RFC3339)
		pubStr = &s
	}
	if e.ValidUntil != nil {
		s := e.ValidUntil.UTC().Format(time.RFC3339)
		validStr = &s
	}

	capStr := e.CapturedAt.UTC().Format(time.RFC3339)
	createdStr := e.CreatedAt.UTC().Format(time.RFC3339)

	_, err := r.db.RawDB().ExecContext(ctx, query,
		e.ID, e.SourceID, e.ContentSnippet, e.ContentHash, e.CaptureMethod, e.DataVersion,
		pubStr, capStr, validStr, createdStr,
	)
	return err
}

// GetEvidence retrieves evidence by ID.
func (r *KnowledgeRepository) GetEvidence(ctx context.Context, id string) (*knowledge.Evidence, error) {
	query := `
	SELECT id, source_id, content_snippet, content_hash, capture_method, data_version, published_at, captured_at, valid_until, created_at
	FROM evidence WHERE id = ?`

	e := &knowledge.Evidence{}
	var pubStr, validStr sql.NullString
	var capStr, createdStr string

	err := r.db.RawDB().QueryRowContext(ctx, query, id).Scan(
		&e.ID, &e.SourceID, &e.ContentSnippet, &e.ContentHash, &e.CaptureMethod, &e.DataVersion,
		&pubStr, &capStr, &validStr, &createdStr,
	)
	if err != nil {
		return nil, err
	}

	if pubStr.Valid {
		t, _ := time.Parse(time.RFC3339, pubStr.String)
		e.PublishedAt = &t
	}
	if validStr.Valid {
		t, _ := time.Parse(time.RFC3339, validStr.String)
		e.ValidUntil = &t
	}
	e.CapturedAt, _ = time.Parse(time.RFC3339, capStr)
	e.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)

	return e, nil
}

// CreateClaim creates a new verifiable claim.
func (r *KnowledgeRepository) CreateClaim(ctx context.Context, c *knowledge.Claim) error {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = c.CreatedAt
	}
	if c.CurrentVersion == 0 {
		c.CurrentVersion = 1
	}

	query := `
	INSERT INTO claims (id, statement, assertion_status, is_knowledge, current_version, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)`

	createdStr := c.CreatedAt.UTC().Format(time.RFC3339)
	updatedStr := c.UpdatedAt.UTC().Format(time.RFC3339)
	isKnowledgeInt := 0
	if c.IsKnowledge {
		isKnowledgeInt = 1
	}

	_, err := r.db.RawDB().ExecContext(ctx, query,
		c.ID, c.Statement, c.AssertionStatus, isKnowledgeInt, c.CurrentVersion, createdStr, updatedStr,
	)
	return err
}

// GetClaim retrieves a claim by ID.
func (r *KnowledgeRepository) GetClaim(ctx context.Context, claimID string) (*knowledge.Claim, error) {
	row := r.db.RawDB().QueryRowContext(ctx, `
		SELECT id, statement, assertion_status, is_knowledge, current_version, created_at, updated_at
		FROM claims WHERE id = ?
	`, claimID)
	c := &knowledge.Claim{}
	var statusStr, createdStr, updatedStr string
	var isKnow int
	if err := row.Scan(&c.ID, &c.Statement, &statusStr, &isKnow, &c.CurrentVersion, &createdStr, &updatedStr); err != nil {
		return nil, err
	}
	c.AssertionStatus = knowledge.AssertionStatus(statusStr)
	c.IsKnowledge = isKnow == 1
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return c, nil
}

// ReviseClaim updates a claim statement atomically and appends to claim_revisions.
func (r *KnowledgeRepository) ReviseClaim(ctx context.Context, claimID, newStatement, reason, decisionID string) error {
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		// 1. Get current claim
		var oldStatement string
		var curVer int
		err := tx.QueryRowContext(ctx, "SELECT statement, current_version FROM claims WHERE id = ?", claimID).Scan(&oldStatement, &curVer)
		if err != nil {
			return err
		}

		nextVer := curVer + 1
		now := time.Now().UTC()
		nowStr := now.Format(time.RFC3339)

		// 2. Update claims table
		_, err = tx.ExecContext(ctx, "UPDATE claims SET statement = ?, current_version = ?, updated_at = ? WHERE id = ?", newStatement, nextVer, nowStr, claimID)
		if err != nil {
			return err
		}

		// 3. Append to claim_revisions
		revID := fmt.Sprintf("%s-rev-%d", claimID, nextVer)
		var decID *string
		if decisionID != "" {
			decID = &decisionID
		}

		revQuery := `
		INSERT INTO claim_revisions (id, claim_id, revision_number, previous_statement, new_statement, reason, revised_decision_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

		_, err = tx.ExecContext(ctx, revQuery, revID, claimID, nextVer, oldStatement, newStatement, reason, decID, nowStr)
		return err
	})
}

// LinkClaimEvidence connects a Claim to Evidence with support type and confidence weight.
func (r *KnowledgeRepository) LinkClaimEvidence(ctx context.Context, link *knowledge.ClaimEvidenceLink) error {
	query := `
	INSERT INTO claim_evidence_links (claim_id, evidence_id, support_type, confidence_weight)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(claim_id, evidence_id) DO UPDATE SET support_type = excluded.support_type, confidence_weight = excluded.confidence_weight`

	_, err := r.db.RawDB().ExecContext(ctx, query, link.ClaimID, link.EvidenceID, link.SupportType, link.ConfidenceWeight)
	return err
}

// ListClaims retrieves claims optionally filtered by assertion_status (e.g. CONFLICTED, ACCEPTED).
func (r *KnowledgeRepository) ListClaims(ctx context.Context, status knowledge.AssertionStatus, limit int) ([]*knowledge.Claim, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, statement, assertion_status, is_knowledge, current_version, created_at, updated_at
		FROM claims
	`
	var args []any
	if status != "" {
		query += ` WHERE assertion_status = ?`
		args = append(args, string(status))
	}
	query += ` ORDER BY updated_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.RawDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*knowledge.Claim
	for rows.Next() {
		c := &knowledge.Claim{}
		var statusStr, createdStr, updatedStr string
		var isKnow int
		if err := rows.Scan(&c.ID, &c.Statement, &statusStr, &isKnow, &c.CurrentVersion, &createdStr, &updatedStr); err != nil {
			return nil, err
		}
		c.AssertionStatus = knowledge.AssertionStatus(statusStr)
		c.IsKnowledge = isKnow == 1
		c.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpdateClaimStatus updates a claim's assertion_status and is_knowledge flag (e.g. after human reconciliation).
func (r *KnowledgeRepository) UpdateClaimStatus(ctx context.Context, claimID string, status knowledge.AssertionStatus, isKnowledge bool) error {
	isKnowInt := 0
	if isKnowledge {
		isKnowInt = 1
	}
	_, err := r.db.RawDB().ExecContext(ctx,
		`UPDATE claims SET assertion_status = ?, is_knowledge = ?, updated_at = ? WHERE id = ?`,
		string(status), isKnowInt, time.Now().UTC().Format(time.RFC3339), claimID,
	)
	return err
}
