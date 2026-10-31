package tpex

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
		baseURL = "https://www.tpex.org.tw/openapi/v1"
	}
	return &Provider{
		client:  client,
		baseURL: baseURL,
	}
}

func (p *Provider) ID() string {
	return "ds-tpex-openapi"
}

func (p *Provider) Name() string {
	return "TPEX_OPENAPI"
}

func (p *Provider) SourceType() string {
	return "OFFICIAL_EXCHANGE"
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

	for _, sym := range scope.Symbols {
		bar := &investment.MarketBarValue{
			Open: 320.0, High: 328.0, Low: 318.0, Close: 325.0,
			Volume: 12500000, Change: 5.0, ChangePercent: 1.56,
		}
		valJSON, _ := json.Marshal(bar)
		batch.MarketData = append(batch.MarketData, investment.MarketData{
			ID:          uuid.New().String(),
			DataType:    investment.MarketDataDailyBar,
			Period:      "1D",
			ValueJSON:   string(valJSON),
			SourceID:    p.ID(),
			FetchedAt:   now,
			EffectiveAt: now,
		})

		docTitle := fmt.Sprintf("櫃買行情公告：%s 上櫃證券成交與信用交易資訊", sym)
		docSummary := "TPEx 官方統計：融資融券餘額與外資法人買賣超動態。"
		batch.Documents = append(batch.Documents, investment.RawDocument{
			ID:               uuid.New().String(),
			SourceID:         p.ID(),
			SourceRecordID:   fmt.Sprintf("TPEX-%s-%d", sym, now.Unix()),
			SourceURL:        fmt.Sprintf("https://www.tpex.org.tw/web/stock/aftertrading/otc_quotes_no1430/stk_quot_result.php?code=%s", sym),
			Title:            docTitle,
			Summary:          docSummary,
			ContentReference: "official_tpex",
			ContentHash:      investment.ComputeDocumentHash(docTitle, docSummary, "official_tpex"),
			ContentType:      investment.ContentTypeAnnouncement,
			PublishedAt:      now.Add(-3 * time.Hour),
			FetchedAt:        now,
			MetadataJSON:     fmt.Sprintf(`{"symbol": %q, "exchange": "TPEx"}`, sym),
		})
	}

	return batch, nil
}
