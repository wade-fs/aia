package investment

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/wade/aia/internal/domain"
)

type SnapshotType string

const (
	SnapshotTypeResearch   SnapshotType = "RESEARCH"
	SnapshotTypeValuation  SnapshotType = "VALUATION"
	SnapshotTypeMacroScope SnapshotType = "MACRO_SCOPE"
)

// DataSnapshot represents an immutable point-in-time state of market data and news for Research Agents (AIA-05 §10, §18.2).
// Research Agents must read strictly from a Snapshot to ensure auditability and deterministic reasoning.
type DataSnapshot struct {
	ID                    domain.ID    `json:"id"`
	SnapshotType          SnapshotType `json:"snapshot_type"`
	DataCutoffAt          time.Time    `json:"data_cutoff_at"`
	InstrumentScopeJSON   string       `json:"instrument_scope_json"` // Array of Symbol/Instrument IDs
	SourceVersionsJSON    string       `json:"source_versions_json"`  // Data source versions/status at cutoff
	ValidationSummaryJSON string       `json:"validation_summary_json"`
	CreatedAt             time.Time    `json:"created_at"`
}

type SnapshotItemType string

const (
	SnapshotItemMarketData SnapshotItemType = "MARKET_DATA"
	SnapshotItemDocument   SnapshotItemType = "RAW_DOCUMENT"
)

// SnapshotItem is an immutable payload linked to a DataSnapshot.
type SnapshotItem struct {
	ID          domain.ID        `json:"id"`
	SnapshotID  domain.ID        `json:"snapshot_id"`
	ItemType    SnapshotItemType `json:"item_type"`
	ReferenceID domain.ID        `json:"reference_id"` // MarketData ID or RawDocument ID
	PayloadJSON string           `json:"payload_json"`
	CreatedAt   time.Time        `json:"created_at"`
}

type SnapshotValidationSummary struct {
	Passed          bool     `json:"passed"`
	MarketDataCount int      `json:"market_data_count"`
	DocumentCount   int      `json:"document_count"`
	Issues          []string `json:"issues,omitempty"`
}

func (s *DataSnapshot) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("snapshot id cannot be empty")
	}
	if s.DataCutoffAt.IsZero() {
		return fmt.Errorf("data_cutoff_at must be provided")
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}
	var valSum SnapshotValidationSummary
	if s.ValidationSummaryJSON != "" {
		if err := json.Unmarshal([]byte(s.ValidationSummaryJSON), &valSum); err != nil {
			return fmt.Errorf("invalid validation_summary_json: %w", err)
		}
	}
	return nil
}
