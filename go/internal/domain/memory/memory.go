package memory

import (
	"time"

	"github.com/wade/aia/internal/domain"
)

type MemoryType string

const (
	TypeFact       MemoryType = "FACT"
	TypeDecision   MemoryType = "DECISION"
	TypePreference MemoryType = "PREFERENCE"
	TypeRule       MemoryType = "RULE"
	TypeLearning   MemoryType = "LEARNING"
)

type MemoryScope string

const (
	ScopeGlobal  MemoryScope = "GLOBAL"
	ScopeProject MemoryScope = "PROJECT"
	ScopeAgent   MemoryScope = "AGENT"
)

type Memory struct {
	ID                domain.ID   `json:"id"`
	Type              MemoryType  `json:"type"`
	Scope             MemoryScope `json:"scope"`
	ProjectID         domain.ID   `json:"project_id"`
	AgentID           domain.ID   `json:"agent_id"`
	Title             string      `json:"title"`
	Content           string      `json:"content"`
	SourceType        string      `json:"source_type"`
	SourceID          domain.ID   `json:"source_id"`
	Confidence        float64     `json:"confidence"`
	Status            string      `json:"status"`
	Version           int         `json:"version"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
	LastUsedAt        *time.Time  `json:"last_used_at"`
	ExpiresAt         *time.Time  `json:"expires_at"`
	ProvenanceType    string      `json:"provenance_type,omitempty"`
	ProvenanceRefType string      `json:"provenance_ref_type,omitempty"`
	ProvenanceRefID   *string     `json:"provenance_ref_id,omitempty"`
	ProvenanceStatus  string      `json:"provenance_status,omitempty"`
}

type ReviewStatus string

const (
	ReviewPending  ReviewStatus = "PENDING"
	ReviewApproved ReviewStatus = "APPROVED"
	ReviewRejected ReviewStatus = "REJECTED"
)

type MemoryCandidate struct {
	ID           domain.ID    `json:"id"`
	Type         MemoryType   `json:"type"`
	Scope        MemoryScope  `json:"scope"`
	ProjectID    domain.ID    `json:"project_id"`
	AgentID      domain.ID    `json:"agent_id"`
	Title        string       `json:"title"`
	Content      string       `json:"content"`
	SourceType   string       `json:"source_type"`
	SourceID     domain.ID    `json:"source_id"`
	Confidence   float64      `json:"confidence"`
	ReviewStatus ReviewStatus `json:"review_status"`
	CreatedAt    time.Time    `json:"created_at"`
	ReviewedAt   *time.Time   `json:"reviewed_at"`
}
