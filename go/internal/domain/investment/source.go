package investment

import (
	"fmt"
	"time"

	"github.com/wade/aia/internal/domain"
)

type SourceStatus string

const (
	SourceStatusActive   SourceStatus = "ACTIVE"
	SourceStatusInactive SourceStatus = "INACTIVE"
)

type SourceHealth string

const (
	SourceHealthHealthy  SourceHealth = "HEALTHY"
	SourceHealthDegraded SourceHealth = "DEGRADED"
	SourceHealthDown     SourceHealth = "DOWN"
)

// DataSource represents a market or document data origin (e.g. TWSE, TPEx, FRED, SEC, MOPS) (AIA-05 §6, §18.1).
type DataSource struct {
	ID                 domain.ID    `json:"id"`
	SourceName         string       `json:"source_name"`
	SourceType         string       `json:"source_type"` // e.g. "OFFICIAL_EXCHANGE", "GOVERNMENT_MOPS", "CENTRAL_BANK", "REGULATOR", "NEWS_RSS"
	BaseURL            string       `json:"base_url"`
	DocumentationURL   string       `json:"documentation_url"`
	AuthenticationType string       `json:"authentication_type"` // NONE, API_KEY, OAUTH
	ConfigurationJSON  string       `json:"configuration_json"`
	Status             SourceStatus `json:"status"`
	HealthStatus       SourceHealth `json:"health_status"`
	LastSuccessAt      *time.Time   `json:"last_success_at,omitempty"`
	LastFailureAt      *time.Time   `json:"last_failure_at,omitempty"`
	CreatedAt          time.Time    `json:"created_at"`
	UpdatedAt          time.Time    `json:"updated_at"`
}

// Validate checks whether essential metadata of a DataSource is valid.
func (s *DataSource) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("data source id cannot be empty")
	}
	if s.SourceName == "" {
		return fmt.Errorf("data source name cannot be empty")
	}
	if s.SourceType == "" {
		return fmt.Errorf("data source type cannot be empty")
	}
	if s.Status == "" {
		s.Status = SourceStatusActive
	}
	if s.HealthStatus == "" {
		s.HealthStatus = SourceHealthHealthy
	}
	return nil
}
