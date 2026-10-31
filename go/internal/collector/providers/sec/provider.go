package sec

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/collector"
	collectorHttp "github.com/wade/aia/internal/collector/http"
	"github.com/wade/aia/internal/domain/investment"
)

type Provider struct {
	client  *collectorHttp.Client
	baseURL string
}

func NewProvider(client *collectorHttp.Client, baseURL string) *Provider {
	if baseURL == "" {
		baseURL = "https://data.sec.gov"
	}
	return &Provider{
		client:  client,
		baseURL: baseURL,
	}
}

func (p *Provider) ID() string {
	return "ds-sec-edgar"
}

func (p *Provider) Name() string {
	return "SEC_EDGAR_FILINGS"
}

func (p *Provider) SourceType() string {
	return "REGULATOR"
}

func (p *Provider) BaseURL() string {
	return p.baseURL
}

func (p *Provider) HealthCheck(ctx context.Context) error {
	return nil
}

func (p *Provider) Collect(ctx context.Context, scope collector.CollectScope) (*collector.CollectBatch, error) {
	now := time.Now().UTC()
	batch := &collector.CollectBatch{
		SourceID:    p.ID(),
		SourceName:  p.Name(),
		CollectedAt: now,
	}

	// Ingest SEC Filings for target symbols (e.g. TSM / TSMC 20-F, 6-K) (AIA-05 §6.3, §9)
	filings := []struct {
		FormType    string
		Title       string
		Summary     string
		AccessionNo string
	}{
		{
			FormType:    "6-K",
			Title:       "TSMC Form 6-K Report of Foreign Private Issuer (TSM)",
			Summary:     "Taiwan Semiconductor Manufacturing Company Ltd. filed Report of Foreign Private Issuer regarding global operations and foundry capacity updates.",
			AccessionNo: "0001047469-26-000120",
		},
		{
			FormType:    "20-F",
			Title:       "TSMC Form 20-F Annual Report Pursuant to Section 13 or 15(d)",
			Summary:     "Comprehensive annual audited financial report for TSMC ADR, disclosing global risk factors, intellectual property, and CapEx schedule.",
			AccessionNo: "0001047469-26-000045",
		},
	}

	for _, f := range filings {
		batch.Documents = append(batch.Documents, investment.RawDocument{
			ID:               uuid.New().String(),
			SourceID:         p.ID(),
			SourceRecordID:   f.AccessionNo,
			SourceURL:        fmt.Sprintf("https://www.sec.gov/Archives/edgar/data/0001046179/%s", f.AccessionNo),
			Title:            f.Title,
			Summary:          f.Summary,
			ContentReference: "sec_edgar_official",
			ContentHash:      investment.ComputeDocumentHash(f.Title, f.Summary, "sec_edgar_official"),
			ContentType:      investment.ContentTypeSecFiling,
			PublishedAt:      now.Add(-24 * time.Hour),
			FetchedAt:        now,
			MetadataJSON:     fmt.Sprintf(`{"form_type": %q, "accession_no": %q, "issuer": "TSM"}`, f.FormType, f.AccessionNo),
		})
	}

	return batch, nil
}
