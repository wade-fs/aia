package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/agent"
	"github.com/wade/aia/internal/storage/sqlite"
)

type AgentRepository struct {
	db *sqlite.DB
}

func NewAgentRepository(db *sqlite.DB) *AgentRepository {
	return &AgentRepository{db: db}
}

func (r *AgentRepository) Upsert(ctx context.Context, a *agent.Agent) error {
	query := `
		INSERT INTO agents (id, name, description, domain, instructions, capabilities_json, default_profile_id, provider_name, provider_agent_name, enabled, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			domain = excluded.domain,
			instructions = excluded.instructions,
			capabilities_json = excluded.capabilities_json,
			default_profile_id = excluded.default_profile_id,
			provider_name = excluded.provider_name,
			provider_agent_name = excluded.provider_agent_name,
			enabled = excluded.enabled,
			version = excluded.version,
			updated_at = excluded.updated_at
	`
	capsJSON, err := json.Marshal(a.Capabilities)
	if err != nil {
		capsJSON = []byte("[]")
	}

	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	a.UpdatedAt = now

	enabledInt := 0
	if a.Enabled {
		enabledInt = 1
	}

	_, err = r.db.RawDB().ExecContext(
		ctx, query,
		a.ID, a.Name, a.Description, a.Domain, a.Instructions, string(capsJSON),
		a.DefaultProfileID, a.ProviderName, a.ProviderAgentName, enabledInt, a.Version,
		a.CreatedAt.Format(time.RFC3339), a.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to upsert agent: %w", err)
	}
	return nil
}

func (r *AgentRepository) Get(ctx context.Context, id domain.ID) (*agent.Agent, error) {
	query := `
		SELECT id, name, description, domain, instructions, capabilities_json, default_profile_id, provider_name, provider_agent_name, enabled, version, created_at, updated_at
		FROM agents WHERE id = ?
	`
	row := r.db.RawDB().QueryRowContext(ctx, query, id)

	var a agent.Agent
	var capsJSON, createdAt, updatedAt string
	var enabledInt int
	var defProfile, provAgent sql.NullString

	err := row.Scan(&a.ID, &a.Name, &a.Description, &a.Domain, &a.Instructions, &capsJSON, &defProfile, &a.ProviderName, &provAgent, &enabledInt, &a.Version, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	a.Enabled = enabledInt == 1
	if defProfile.Valid {
		a.DefaultProfileID = defProfile.String
	}
	if provAgent.Valid {
		a.ProviderAgentName = provAgent.String
	}
	_ = json.Unmarshal([]byte(capsJSON), &a.Capabilities)
	a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	a.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &a, nil
}

func (r *AgentRepository) List(ctx context.Context) ([]agent.Agent, error) {
	query := `
		SELECT id, name, description, domain, instructions, capabilities_json, default_profile_id, provider_name, provider_agent_name, enabled, version, created_at, updated_at
		FROM agents ORDER BY name ASC
	`
	rows, err := r.db.RawDB().QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}
	defer rows.Close()

	var agents []agent.Agent
	for rows.Next() {
		var a agent.Agent
		var capsJSON, createdAt, updatedAt string
		var enabledInt int
		var defProfile, provAgent sql.NullString

		if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.Domain, &a.Instructions, &capsJSON, &defProfile, &a.ProviderName, &provAgent, &enabledInt, &a.Version, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}
		a.Enabled = enabledInt == 1
		if defProfile.Valid {
			a.DefaultProfileID = defProfile.String
		}
		if provAgent.Valid {
			a.ProviderAgentName = provAgent.String
		}
		_ = json.Unmarshal([]byte(capsJSON), &a.Capabilities)
		a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		a.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		agents = append(agents, a)
	}
	return agents, nil
}

func (r *AgentRepository) SetEnabled(ctx context.Context, id domain.ID, enabled bool) error {
	query := `UPDATE agents SET enabled = ?, updated_at = ? WHERE id = ?`
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.RawDB().ExecContext(ctx, query, enabledInt, now, id)
	if err != nil {
		return fmt.Errorf("failed to update agent enabled: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("agent not found: %s", id)
	}
	return nil
}
