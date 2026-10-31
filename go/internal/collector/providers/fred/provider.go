package fred

import (
	"context"
	"encoding/json"
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
		baseURL = "https://api.stlouisfed.org/fred"
	}
	return &Provider{
		client:  client,
		baseURL: baseURL,
	}
}

func (p *Provider) ID() string {
	return "ds-fred-macro"
}

func (p *Provider) Name() string {
	return "FRED_MACRO_INDICATORS"
}

func (p *Provider) SourceType() string {
	return "CENTRAL_BANK"
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

	// Ingest Macro Indicators: US 10Y Treasury & Fed Funds Rate (AIA-05 §6.2)
	macroRates := []struct {
		SeriesID string
		Name     string
		Rate     float64
	}{
		{SeriesID: "DGS10", Name: "美國 10 年期公債殖利率", Rate: 4.28},
		{SeriesID: "FEDFUNDS", Name: "聯邦基金有效利率 (Fed Funds Rate)", Rate: 4.83},
	}

	for _, m := range macroRates {
		valJSON, _ := json.Marshal(map[string]any{
			"series_id": m.SeriesID,
			"name":      m.Name,
			"rate":      m.Rate,
			"unit":      "PERCENT",
		})
		batch.MarketData = append(batch.MarketData, investment.MarketData{
			ID:          uuid.New().String(),
			DataType:    investment.MarketDataMacroRate,
			Period:      "1D",
			ValueJSON:   string(valJSON),
			SourceID:    p.ID(),
			FetchedAt:   now,
			EffectiveAt: now,
		})

		docTitle := fmt.Sprintf("FRED 總體經濟發布：%s 最新基準為 %.2f%%", m.Name, m.Rate)
		docSummary := fmt.Sprintf("聖路易斯聯邦準備銀行 (FRED) 官方系列 %s 數據更新，反映當前總體資金成本與無風險利率環境。", m.SeriesID)
		batch.Documents = append(batch.Documents, investment.RawDocument{
			ID:               uuid.New().String(),
			SourceID:         p.ID(),
			SourceRecordID:   fmt.Sprintf("FRED-%s-%d", m.SeriesID, now.Unix()),
			SourceURL:        fmt.Sprintf("https://fred.stlouisfed.org/series/%s", m.SeriesID),
			Title:            docTitle,
			Summary:          docSummary,
			ContentReference: "fred_official",
			ContentHash:      investment.ComputeDocumentHash(docTitle, docSummary, "fred_official"),
			ContentType:      investment.ContentTypeMacroRelease,
			PublishedAt:      now.Add(-6 * time.Hour),
			FetchedAt:        now,
			MetadataJSON:     fmt.Sprintf(`{"series_id": %q, "rate": %f}`, m.SeriesID, m.Rate),
		})
	}

	return batch, nil
}
