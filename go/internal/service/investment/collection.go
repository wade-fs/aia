package investment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/collector"
	collectorHttp "github.com/wade/aia/internal/collector/http"
	"github.com/wade/aia/internal/collector/providers/fred"
	"github.com/wade/aia/internal/collector/providers/sec"
	"github.com/wade/aia/internal/collector/providers/tpex"
	"github.com/wade/aia/internal/collector/providers/twse"
	"github.com/wade/aia/internal/domain/investment"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type CollectionSummary struct {
	SourceID          string   `json:"source_id"`
	SourceName        string   `json:"source_name"`
	Success           bool     `json:"success"`
	DocumentsSaved    int      `json:"documents_saved"`
	DuplicatesSkipped int      `json:"duplicates_skipped"`
	MarketDataSaved   int      `json:"market_data_saved"`
	ValidationErrors  []string `json:"validation_errors,omitempty"`
	ErrorMessage      string   `json:"error_message,omitempty"`
}

// DataCollectorService manages multi-provider data ingestion, deduplication, validation,
// and immutable Snapshot generation for Research Agents (AIA-05 §6, §8, §10).
type DataCollectorService struct {
	dataRepo     *repository.InvestmentDataRepository
	invRepo      *repository.InvestmentRepository
	collectorMgr *collector.Manager
}

func NewDataCollectorService(
	dataRepo *repository.InvestmentDataRepository,
	invRepo *repository.InvestmentRepository,
) *DataCollectorService {
	mgr := collector.NewManager()

	// Provision standard HTTP client
	client := collectorHttp.NewClient(10*time.Second, 5, "")

	// Register official providers: TWSE, TPEx, FRED, SEC
	_ = mgr.Register(twse.NewProvider(client, ""))
	_ = mgr.Register(tpex.NewProvider(client, ""))
	_ = mgr.Register(fred.NewProvider(client, ""))
	_ = mgr.Register(sec.NewProvider(client, ""))

	return &DataCollectorService{
		dataRepo:     dataRepo,
		invRepo:      invRepo,
		collectorMgr: mgr,
	}
}

func (s *DataCollectorService) RegisterProvider(p collector.Provider) error {
	return s.collectorMgr.Register(p)
}

// InitDefaultSources registers and seeds the official data source records in DB.
func (s *DataCollectorService) InitDefaultSources(ctx context.Context) error {
	providers := s.collectorMgr.Providers()
	for _, p := range providers {
		ds := &investment.DataSource{
			ID:                 p.ID(),
			SourceName:         p.Name(),
			SourceType:         p.SourceType(),
			BaseURL:            p.BaseURL(),
			AuthenticationType: "NONE",
			Status:             investment.SourceStatusActive,
			HealthStatus:       investment.SourceHealthHealthy,
			ConfigurationJSON:  `{"rate_limit_rpm": 60}`,
		}
		if err := s.dataRepo.SaveDataSource(ctx, ds); err != nil {
			return fmt.Errorf("failed to save data source %s: %w", p.Name(), err)
		}
	}
	return nil
}

func (s *DataCollectorService) ListDataSources(ctx context.Context) ([]investment.DataSource, error) {
	_ = s.InitDefaultSources(ctx)
	return s.dataRepo.ListDataSources(ctx)
}

// CollectFromProvider executes a collection run against a specific provider with deduplication and validation.
func (s *DataCollectorService) CollectFromProvider(ctx context.Context, providerID string, scope collector.CollectScope) (*CollectionSummary, error) {
	_ = s.InitDefaultSources(ctx)

	prov, ok := s.collectorMgr.Get(providerID)
	if !ok {
		return nil, fmt.Errorf("provider %s not registered", providerID)
	}

	summary := &CollectionSummary{
		SourceID:   prov.ID(),
		SourceName: prov.Name(),
	}

	// 1. Fetch raw batch from external provider
	batch, err := prov.Collect(ctx, scope)
	if err != nil {
		summary.Success = false
		summary.ErrorMessage = err.Error()
		// Mark source degraded; existing local records are NEVER destroyed
		_ = s.dataRepo.UpdateDataSourceHealth(ctx, prov.ID(), investment.SourceHealthDegraded, false)
		slog.Warn("data collection error from external provider", "source", prov.Name(), "error", err)
		return summary, nil
	}

	summary.Success = true
	_ = s.dataRepo.UpdateDataSourceHealth(ctx, prov.ID(), investment.SourceHealthHealthy, true)

	// 2. Ingest and Validate Market Data
	for _, md := range batch.MarketData {
		// Link Instrument ID if symbols given
		if md.InstrumentID == "" && len(scope.Symbols) > 0 {
			inst, err := s.invRepo.GetInstrumentBySymbol(ctx, "", scope.Symbols[0])
			if err == nil && inst != nil {
				md.InstrumentID = inst.ID
			}
		}

		if err := md.Validate(); err != nil {
			summary.ValidationErrors = append(summary.ValidationErrors, fmt.Sprintf("market_data invalid: %v", err))
			continue
		}

		if err := s.dataRepo.SaveMarketData(ctx, &md); err == nil {
			summary.MarketDataSaved++
		}
	}

	// 3. Ingest, Validate, and Deduplicate Raw Documents
	for _, doc := range batch.Documents {
		if doc.InstrumentID == "" && len(scope.Symbols) > 0 {
			inst, err := s.invRepo.GetInstrumentBySymbol(ctx, "", scope.Symbols[0])
			if err == nil && inst != nil {
				doc.InstrumentID = inst.ID
			}
		}

		if err := doc.Validate(); err != nil {
			summary.ValidationErrors = append(summary.ValidationErrors, fmt.Sprintf("document invalid: %v", err))
			continue
		}

		isDup, err := s.dataRepo.SaveRawDocument(ctx, &doc)
		if err != nil {
			summary.ValidationErrors = append(summary.ValidationErrors, fmt.Sprintf("save doc error: %v", err))
			continue
		}
		if isDup {
			summary.DuplicatesSkipped++
		} else {
			summary.DocumentsSaved++
		}
	}

	return summary, nil
}

// CollectAll triggers a sweep across all registered data providers.
func (s *DataCollectorService) CollectAll(ctx context.Context, scope collector.CollectScope) ([]CollectionSummary, error) {
	providers := s.collectorMgr.Providers()
	var summaries []CollectionSummary
	for _, p := range providers {
		res, err := s.CollectFromProvider(ctx, p.ID(), scope)
		if err == nil && res != nil {
			summaries = append(summaries, *res)
		}
	}
	return summaries, nil
}

// CreateResearchSnapshot creates an immutable point-in-time Research Snapshot (AIA-05 §10).
// Research Agents must read strictly from a Snapshot to ensure deterministic analysis.
func (s *DataCollectorService) CreateResearchSnapshot(
	ctx context.Context,
	symbol string,
	cutoff time.Time,
) (*investment.DataSnapshot, error) {
	if cutoff.IsZero() {
		cutoff = time.Now().UTC()
	}

	inst, err := s.invRepo.GetInstrumentBySymbol(ctx, "", symbol)
	if err != nil {
		return nil, fmt.Errorf("instrument %s not found: %w", symbol, err)
	}

	// 1. Fetch market data effective at or before cutoff
	bars, err := s.dataRepo.ListHistoricalMarketData(ctx, inst.ID, 10)
	if err != nil {
		return nil, err
	}

	// 2. Fetch raw documents published at or before cutoff
	docs, err := s.dataRepo.ListRawDocuments(ctx, inst.ID, 10)
	if err != nil {
		return nil, err
	}

	// 3. Package Snapshot Items
	var items []investment.SnapshotItem
	marketCount := 0
	docCount := 0

	for _, b := range bars {
		if !b.EffectiveAt.After(cutoff) {
			val, _ := json.Marshal(b)
			items = append(items, investment.SnapshotItem{
				ID:          uuid.New().String(),
				ItemType:    investment.SnapshotItemMarketData,
				ReferenceID: b.ID,
				PayloadJSON: string(val),
				CreatedAt:   time.Now().UTC(),
			})
			marketCount++
		}
	}

	for _, d := range docs {
		if !d.PublishedAt.After(cutoff) {
			val, _ := json.Marshal(d)
			items = append(items, investment.SnapshotItem{
				ID:          uuid.New().String(),
				ItemType:    investment.SnapshotItemDocument,
				ReferenceID: d.ID,
				PayloadJSON: string(val),
				CreatedAt:   time.Now().UTC(),
			})
			docCount++
		}
	}

	// 4. Construct Validation Summary
	valSum := investment.SnapshotValidationSummary{
		Passed:          true,
		MarketDataCount: marketCount,
		DocumentCount:   docCount,
	}
	if marketCount == 0 && docCount == 0 {
		valSum.Passed = false
		valSum.Issues = append(valSum.Issues, "snapshot contains 0 items prior to cutoff")
	}
	valJSON, _ := json.Marshal(valSum)

	scopeJSON, _ := json.Marshal([]string{symbol})

	snapshot := &investment.DataSnapshot{
		ID:                    uuid.New().String(),
		SnapshotType:          investment.SnapshotTypeResearch,
		DataCutoffAt:          cutoff,
		InstrumentScopeJSON:   string(scopeJSON),
		SourceVersionsJSON:    `{"twse": "v1", "tpex": "v1", "fred": "v1", "sec": "v1"}`,
		ValidationSummaryJSON: string(valJSON),
		CreatedAt:             time.Now().UTC(),
	}

	if err := s.dataRepo.CreateDataSnapshot(ctx, snapshot, items); err != nil {
		return nil, fmt.Errorf("failed to create data snapshot: %w", err)
	}

	return snapshot, nil
}

func (s *DataCollectorService) GetDataSnapshot(ctx context.Context, id string) (*investment.DataSnapshot, []investment.SnapshotItem, error) {
	snap, err := s.dataRepo.GetDataSnapshot(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	items, err := s.dataRepo.ListSnapshotItems(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return snap, items, nil
}
