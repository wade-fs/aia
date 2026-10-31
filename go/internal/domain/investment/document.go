package investment

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/wade/aia/internal/domain"
)

type DocumentContentType string

const (
	ContentTypeNews         DocumentContentType = "NEWS"
	ContentTypeAnnouncement DocumentContentType = "ANNOUNCEMENT"
	ContentTypeReport       DocumentContentType = "REPORT"
	ContentTypeSecFiling    DocumentContentType = "SEC_FILING"
	ContentTypeMacroRelease DocumentContentType = "MACRO_RELEASE"
)

// RawDocument represents external unstructured text collected from official sources or feeds (AIA-05 §8, §18.2).
type RawDocument struct {
	ID               domain.ID           `json:"id"`
	SourceID         domain.ID           `json:"source_id"`
	InstrumentID     domain.ID           `json:"instrument_id,omitempty"`
	SourceRecordID   string              `json:"source_record_id"`
	SourceURL        string              `json:"source_url"`
	Title            string              `json:"title"`
	Summary          string              `json:"summary"`
	ContentReference string              `json:"content_reference"`
	ContentHash      string              `json:"content_hash"`
	ContentType      DocumentContentType `json:"content_type"`
	PublishedAt      time.Time           `json:"published_at"`
	FetchedAt        time.Time           `json:"fetched_at"`
	MetadataJSON     string              `json:"metadata_json"`
}

// ComputeDocumentHash computes a deterministic SHA-256 hash across title, summary, and content reference for deduplication.
func ComputeDocumentHash(title, summary, reference string) string {
	payload := strings.TrimSpace(title) + "|" + strings.TrimSpace(summary) + "|" + strings.TrimSpace(reference)
	h := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(h[:])
}

// Validate ensures basic fields and temporal consistency (AIA-05 §8.1).
func (d *RawDocument) Validate() error {
	if d.ID == "" {
		return fmt.Errorf("document id cannot be empty")
	}
	if d.SourceID == "" {
		return fmt.Errorf("document source_id cannot be empty")
	}
	if strings.TrimSpace(d.Title) == "" {
		return fmt.Errorf("document title cannot be empty")
	}
	if d.PublishedAt.IsZero() {
		return fmt.Errorf("document published_at must be provided")
	}
	// Allow up to 10 minutes clock drift for published_at against future timestamps
	if d.PublishedAt.After(time.Now().Add(10 * time.Minute)) {
		return fmt.Errorf("document published_at %v is in the future", d.PublishedAt)
	}
	if d.ContentHash == "" {
		d.ContentHash = ComputeDocumentHash(d.Title, d.Summary, d.ContentReference)
	}
	if d.ContentType == "" {
		d.ContentType = ContentTypeNews
	}
	if d.FetchedAt.IsZero() {
		d.FetchedAt = time.Now().UTC()
	}
	return nil
}
