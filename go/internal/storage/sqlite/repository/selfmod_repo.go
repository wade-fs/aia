package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wade/aia/internal/domain/selfmod"
	"github.com/wade/aia/internal/storage/sqlite"
)

type SelfModRepository struct {
	db *sqlite.DB
}

func NewSelfModRepository(db *sqlite.DB) *SelfModRepository {
	return &SelfModRepository{db: db}
}

func (r *SelfModRepository) Create(ctx context.Context, p *selfmod.SelfModificationProposal) error {
	query := `
		INSERT INTO self_modification_proposals (
			id, title, description, rationale, target_files_json, patches_json,
			scope_level, requires_approval, approval_id, status, branch_name,
			pre_commit_sha, post_commit_sha, verification_output, rca_log,
			created_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}

	targetFilesJSON, _ := json.Marshal(p.TargetFiles)
	patchesJSON, _ := json.Marshal(p.Patches)

	var completedAt *string
	if p.CompletedAt != nil {
		s := p.CompletedAt.UTC().Format(time.RFC3339)
		completedAt = &s
	}

	_, err := r.db.RawDB().ExecContext(ctx, query,
		p.ID, p.Title, p.Description, p.Rationale, string(targetFilesJSON), string(patchesJSON),
		string(p.ScopeLevel), p.RequiresApproval, p.ApprovalID, string(p.Status), p.BranchName,
		p.PreCommitSHA, p.PostCommitSHA, p.VerificationOutput, p.RCALog,
		p.CreatedAt.UTC().Format(time.RFC3339), completedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert selfmod proposal: %w", err)
	}
	return nil
}

func (r *SelfModRepository) Update(ctx context.Context, p *selfmod.SelfModificationProposal) error {
	query := `
		UPDATE self_modification_proposals SET
			title = ?, description = ?, rationale = ?, target_files_json = ?, patches_json = ?,
			scope_level = ?, requires_approval = ?, approval_id = ?, status = ?, branch_name = ?,
			pre_commit_sha = ?, post_commit_sha = ?, verification_output = ?, rca_log = ?,
			completed_at = ?
		WHERE id = ?
	`
	targetFilesJSON, _ := json.Marshal(p.TargetFiles)
	patchesJSON, _ := json.Marshal(p.Patches)

	var completedAt *string
	if p.CompletedAt != nil {
		s := p.CompletedAt.UTC().Format(time.RFC3339)
		completedAt = &s
	}

	res, err := r.db.RawDB().ExecContext(ctx, query,
		p.Title, p.Description, p.Rationale, string(targetFilesJSON), string(patchesJSON),
		string(p.ScopeLevel), p.RequiresApproval, p.ApprovalID, string(p.Status), p.BranchName,
		p.PreCommitSHA, p.PostCommitSHA, p.VerificationOutput, p.RCALog,
		completedAt, p.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update selfmod proposal: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return selfmod.ErrProposalNotFound
	}
	return nil
}

func (r *SelfModRepository) Get(ctx context.Context, id string) (*selfmod.SelfModificationProposal, error) {
	query := `
		SELECT id, title, description, rationale, target_files_json, patches_json,
		       scope_level, requires_approval, approval_id, status, branch_name,
		       pre_commit_sha, post_commit_sha, verification_output, rca_log,
		       created_at, completed_at
		FROM self_modification_proposals
		WHERE id = ?
	`
	var (
		p                                  selfmod.SelfModificationProposal
		targetFilesJSON, patchesJSON       string
		createdAtStr                       string
		completedAtStr                     sql.NullString
		approvalIDStr                      sql.NullString
		scopeLevelStr, statusStr           string
	)

	err := r.db.RawDB().QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.Title, &p.Description, &p.Rationale, &targetFilesJSON, &patchesJSON,
		&scopeLevelStr, &p.RequiresApproval, &approvalIDStr, &statusStr, &p.BranchName,
		&p.PreCommitSHA, &p.PostCommitSHA, &p.VerificationOutput, &p.RCALog,
		&createdAtStr, &completedAtStr,
	)
	if err == sql.ErrNoRows {
		return nil, selfmod.ErrProposalNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query selfmod proposal: %w", err)
	}

	p.ScopeLevel = selfmod.ModificationLevel(scopeLevelStr)
	p.Status = selfmod.ModificationStatus(statusStr)
	if approvalIDStr.Valid {
		p.ApprovalID = &approvalIDStr.String
	}
	_ = json.Unmarshal([]byte(targetFilesJSON), &p.TargetFiles)
	_ = json.Unmarshal([]byte(patchesJSON), &p.Patches)

	if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
		p.CreatedAt = t
	}
	if completedAtStr.Valid {
		if t, err := time.Parse(time.RFC3339, completedAtStr.String); err == nil {
			p.CompletedAt = &t
		}
	}

	return &p, nil
}

func (r *SelfModRepository) ListByStatus(ctx context.Context, status selfmod.ModificationStatus) ([]*selfmod.SelfModificationProposal, error) {
	query := `
		SELECT id, title, description, rationale, target_files_json, patches_json,
		       scope_level, requires_approval, approval_id, status, branch_name,
		       pre_commit_sha, post_commit_sha, verification_output, rca_log,
		       created_at, completed_at
		FROM self_modification_proposals
		WHERE status = ?
		ORDER BY created_at DESC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query, string(status))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*selfmod.SelfModificationProposal
	for rows.Next() {
		var (
			p                            selfmod.SelfModificationProposal
			targetFilesJSON, patchesJSON string
			createdAtStr                 string
			completedAtStr               sql.NullString
			approvalIDStr                sql.NullString
			scopeLevelStr, statusStr     string
		)
		if err := rows.Scan(
			&p.ID, &p.Title, &p.Description, &p.Rationale, &targetFilesJSON, &patchesJSON,
			&scopeLevelStr, &p.RequiresApproval, &approvalIDStr, &statusStr, &p.BranchName,
			&p.PreCommitSHA, &p.PostCommitSHA, &p.VerificationOutput, &p.RCALog,
			&createdAtStr, &completedAtStr,
		); err != nil {
			return nil, err
		}
		p.ScopeLevel = selfmod.ModificationLevel(scopeLevelStr)
		p.Status = selfmod.ModificationStatus(statusStr)
		if approvalIDStr.Valid {
			p.ApprovalID = &approvalIDStr.String
		}
		_ = json.Unmarshal([]byte(targetFilesJSON), &p.TargetFiles)
		_ = json.Unmarshal([]byte(patchesJSON), &p.Patches)
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			p.CreatedAt = t
		}
		if completedAtStr.Valid {
			if t, err := time.Parse(time.RFC3339, completedAtStr.String); err == nil {
				p.CompletedAt = &t
			}
		}
		list = append(list, &p)
	}
	return list, nil
}
