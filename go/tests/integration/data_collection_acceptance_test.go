package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/collector"
	"github.com/wade/aia/internal/domain/investment"
)

// MockFailingProvider simulates an external website or provider failing (e.g. HTTP 503 or Network Timeout).
type MockFailingProvider struct{}

func (m *MockFailingProvider) ID() string         { return "ds-failing-exchange" }
func (m *MockFailingProvider) Name() string       { return "FAILING_EXCHANGE" }
func (m *MockFailingProvider) SourceType() string { return "OFFICIAL_EXCHANGE" }
func (m *MockFailingProvider) BaseURL() string    { return "https://failing.exchange.local" }
func (m *MockFailingProvider) HealthCheck(ctx context.Context) error {
	return fmt.Errorf("connection refused")
}
func (m *MockFailingProvider) Collect(ctx context.Context, scope collector.CollectScope) (*collector.CollectBatch, error) {
	return nil, fmt.Errorf("remote host unreachable: network timeout")
}

func TestDataCollectionSevenAcceptanceCriteria(t *testing.T) {
	application, _ := createTestApp(t)
	defer application.Close()

	ctx := context.Background()

	// 0. Ensure instrument & profile exist
	_, err := application.InvSvc.EnsureDefaultProfile(ctx)
	if err != nil {
		t.Fatalf("failed to init profile: %v", err)
	}

	// =========================================================================
	// Criterion 1: 能註冊並管理資料來源 (Register & Manage Data Sources)
	// =========================================================================
	t.Run("Criterion 1: Register and Manage Data Sources", func(t *testing.T) {
		sources, err := application.CollectorSvc.ListDataSources(ctx)
		if err != nil {
			t.Fatalf("failed to list data sources: %v", err)
		}
		if len(sources) < 4 {
			t.Fatalf("expected at least 4 default sources (TWSE, TPEx, FRED, SEC), got %d", len(sources))
		}

		// Verify custom data source registration
		customSource := &investment.DataSource{
			ID:                 "ds-custom-mops",
			SourceName:         "CUSTOM_MOPS",
			SourceType:         "GOVERNMENT_MOPS",
			BaseURL:            "https://mops.custom.tw",
			AuthenticationType: "NONE",
			Status:             investment.SourceStatusActive,
			HealthStatus:       investment.SourceHealthHealthy,
			ConfigurationJSON:  `{"rate_limit_rpm": 60}`,
		}
		if err := application.InvDataRepo.SaveDataSource(ctx, customSource); err != nil {
			t.Fatalf("failed to register custom source: %v", err)
		}

		retrieved, err := application.InvDataRepo.GetDataSource(ctx, customSource.ID)
		if err != nil {
			t.Fatalf("failed to retrieve registered source: %v", err)
		}
		if retrieved.SourceName != "CUSTOM_MOPS" {
			t.Errorf("expected CUSTOM_MOPS, got %s", retrieved.SourceName)
		}
	})

	// =========================================================================
	// Criterion 2: 能執行一次資料收集 (Execute a Data Collection Run)
	// =========================================================================
	t.Run("Criterion 2: Execute a Data Collection Run", func(t *testing.T) {
		summary, err := application.CollectorSvc.CollectFromProvider(ctx, "ds-twse-openapi", collector.CollectScope{
			Symbols: []string{"2330"},
			Limit:   5,
		})
		if err != nil {
			t.Fatalf("failed to execute collection run: %v", err)
		}
		if !summary.Success {
			t.Fatalf("expected collection to succeed, got error: %s", summary.ErrorMessage)
		}
		if summary.MarketDataSaved == 0 && summary.DocumentsSaved == 0 {
			t.Fatalf("expected items to be saved in collection run, got %+v", summary)
		}
	})

	// =========================================================================
	// Criterion 3: 能保存原始資料及來源資訊 (Preserve Raw Data & Origin Info)
	// =========================================================================
	t.Run("Criterion 3: Preserve Raw Data and Origin Metadata", func(t *testing.T) {
		inst, err := application.InvRepo.GetInstrumentBySymbol(ctx, "", "2330")
		if err != nil {
			t.Fatalf("failed to get instrument: %v", err)
		}

		// Verify Market Data origin information
		md, err := application.InvDataRepo.GetLatestMarketData(ctx, inst.ID)
		if err != nil {
			t.Fatalf("failed to get market data: %v", err)
		}
		if md.SourceID == "" {
			t.Error("market data source_id must be preserved")
		}
		if md.EffectiveAt.IsZero() || md.FetchedAt.IsZero() {
			t.Error("effective_at and fetched_at timestamps must be preserved")
		}

		var bar investment.MarketBarValue
		if err := json.Unmarshal([]byte(md.ValueJSON), &bar); err != nil || bar.Close <= 0 {
			t.Errorf("raw bar payload not preserved properly: %v", err)
		}

		// Verify Raw Document origin information
		docs, err := application.InvDataRepo.ListRawDocuments(ctx, inst.ID, 5)
		if err != nil || len(docs) == 0 {
			t.Fatalf("expected raw documents to be preserved: %v", err)
		}
		firstDoc := docs[0]
		if firstDoc.SourceID == "" || firstDoc.ContentHash == "" {
			t.Errorf("document missing source_id or content_hash: %+v", firstDoc)
		}
		if firstDoc.SourceURL == "" || firstDoc.PublishedAt.IsZero() {
			t.Errorf("document missing source_url or published_at: %+v", firstDoc)
		}
	})

	// =========================================================================
	// Criterion 4: 能辨識重複資料 (Recognize and Deduplicate Duplicate Data)
	// =========================================================================
	t.Run("Criterion 4: Deduplicate Identical Documents via Content Hash", func(t *testing.T) {
		inst, err := application.InvRepo.GetInstrumentBySymbol(ctx, "", "2330")
		if err != nil {
			t.Fatalf("failed to get instrument: %v", err)
		}

		now := time.Now().UTC()
		title := "台積電公布最新晶圓代工產能利用率公告"
		summary := "受全球 HPC 晶片需求擴張，先進製程稼動率維持滿載。"
		ref := "internal_test_ref"
		contentHash := investment.ComputeDocumentHash(title, summary, ref)

		doc1 := &investment.RawDocument{
			ID:               uuid.New().String(),
			SourceID:         "ds-twse-openapi",
			InstrumentID:     inst.ID,
			SourceRecordID:   "REC-DUP-001",
			SourceURL:        "https://mops.twse.com.tw/dup001",
			Title:            title,
			Summary:          summary,
			ContentReference: ref,
			ContentHash:      contentHash,
			ContentType:      investment.ContentTypeAnnouncement,
			PublishedAt:      now,
			FetchedAt:        now,
		}

		// First save -> should NOT be duplicate
		isDup1, err := application.InvDataRepo.SaveRawDocument(ctx, doc1)
		if err != nil {
			t.Fatalf("first document insert failed: %v", err)
		}
		if isDup1 {
			t.Error("expected first insert to be a new document, got duplicate=true")
		}

		// Second save with identical source_id and content_hash -> MUST be detected as duplicate
		doc2 := &investment.RawDocument{
			ID:               uuid.New().String(),
			SourceID:         "ds-twse-openapi",
			InstrumentID:     inst.ID,
			SourceRecordID:   "REC-DUP-001-AGAIN",
			SourceURL:        "https://mops.twse.com.tw/dup001",
			Title:            title,
			Summary:          summary,
			ContentReference: ref,
			ContentHash:      contentHash,
			ContentType:      investment.ContentTypeAnnouncement,
			PublishedAt:      now,
			FetchedAt:        now,
		}
		isDup2, err := application.InvDataRepo.SaveRawDocument(ctx, doc2)
		if err != nil {
			t.Fatalf("second document insert failed: %v", err)
		}
		if !isDup2 {
			t.Error("expected second insert to be identified as duplicate, got duplicate=false")
		}
	})

	// =========================================================================
	// Criterion 5: 能驗證資料時間與基本欄位 (Validate Timestamps & Essential Fields)
	// =========================================================================
	t.Run("Criterion 5: Validate Timestamps and Field Integrity", func(t *testing.T) {
		inst, _ := application.InvRepo.GetInstrumentBySymbol(ctx, "", "2330")

		// Case A: Future Published Time (Clock error / forgery)
		futureDoc := &investment.RawDocument{
			ID:           uuid.New().String(),
			SourceID:     "ds-twse-openapi",
			InstrumentID: inst.ID,
			Title:        "Future News",
			PublishedAt:  time.Now().Add(2 * time.Hour), // 2 hours into future
		}
		if err := futureDoc.Validate(); err == nil {
			t.Error("expected validation error for future published_at time, got nil")
		}

		// Case B: Empty title
		emptyTitleDoc := &investment.RawDocument{
			ID:          uuid.New().String(),
			SourceID:    "ds-twse-openapi",
			Title:       "   ",
			PublishedAt: time.Now(),
		}
		if err := emptyTitleDoc.Validate(); err == nil {
			t.Error("expected validation error for empty title, got nil")
		}

		// Case C: Negative or invalid market price
		invalidPriceMD := &investment.MarketData{
			ID:           uuid.New().String(),
			InstrumentID: inst.ID,
			SourceID:     "ds-twse-openapi",
			DataType:     investment.MarketDataDailyBar,
			EffectiveAt:  time.Now(),
			ValueJSON:    `{"open": 100, "close": -10, "volume": 500}`,
		}
		if err := invalidPriceMD.Validate(); err == nil {
			t.Error("expected validation error for negative close price, got nil")
		}
	})

	// =========================================================================
	// Criterion 6: 能建立可供 Research Agent 使用的 Snapshot (Immutable Snapshot)
	// =========================================================================
	t.Run("Criterion 6: Build Immutable Research Snapshot", func(t *testing.T) {
		cutoff := time.Now().UTC()

		snap, err := application.CollectorSvc.CreateResearchSnapshot(ctx, "2330", cutoff)
		if err != nil {
			t.Fatalf("failed to create research snapshot: %v", err)
		}
		if snap == nil || snap.ID == "" {
			t.Fatal("expected snapshot to have an ID")
		}

		// Retrieve and verify immutability of items in snapshot
		retrievedSnap, items, err := application.CollectorSvc.GetDataSnapshot(ctx, snap.ID)
		if err != nil {
			t.Fatalf("failed to get snapshot: %v", err)
		}
		if retrievedSnap.DataCutoffAt.IsZero() {
			t.Error("snapshot cutoff time must be preserved")
		}
		if len(items) == 0 {
			t.Error("expected snapshot to contain items collected prior to cutoff")
		}

		var summary investment.SnapshotValidationSummary
		_ = json.Unmarshal([]byte(retrievedSnap.ValidationSummaryJSON), &summary)
		if !summary.Passed {
			t.Errorf("expected snapshot validation summary to pass, got: %+v", summary)
		}
	})

	// =========================================================================
	// Criterion 7: 即使外部網站暫時無法連線，也不會破壞既有資料 (Fault Resilience)
	// =========================================================================
	t.Run("Criterion 7: External Network Failure Does Not Corrupt Existing Data", func(t *testing.T) {
		inst, _ := application.InvRepo.GetInstrumentBySymbol(ctx, "", "2330")

		// Check baseline data exists before failure
		existingBars, err := application.InvDataRepo.ListHistoricalMarketData(ctx, inst.ID, 10)
		if err != nil || len(existingBars) == 0 {
			t.Fatalf("expected existing bars before failure test, got %d", len(existingBars))
		}
		initialCount := len(existingBars)

		// Register a failing provider simulating network outage
		failingProv := &MockFailingProvider{}
		_ = application.CollectorSvc.RegisterProvider(failingProv)
		_ = application.InvDataRepo.SaveDataSource(ctx, &investment.DataSource{
			ID:                 failingProv.ID(),
			SourceName:         failingProv.Name(),
			SourceType:         failingProv.SourceType(),
			BaseURL:            failingProv.BaseURL(),
			AuthenticationType: "NONE",
			Status:             investment.SourceStatusActive,
			HealthStatus:       investment.SourceHealthHealthy,
		})

		// Execute collection against failing provider
		summary, err := application.CollectorSvc.CollectFromProvider(ctx, failingProv.ID(), collector.CollectScope{
			Symbols: []string{"2330"},
		})
		if err != nil {
			t.Fatalf("service should handle error gracefully without crashing: %v", err)
		}
		if summary.Success {
			t.Error("expected failing provider collection to report success=false")
		}

		// Verify: Data source marked DEGRADED
		ds, err := application.InvDataRepo.GetDataSource(ctx, failingProv.ID())
		if err != nil {
			t.Fatalf("failed to get failing data source: %v", err)
		}
		if ds.HealthStatus != investment.SourceHealthDegraded {
			t.Errorf("expected data source health to be DEGRADED, got %s", ds.HealthStatus)
		}

		// Critical check: Existing data was NOT corrupted or wiped
		afterBars, err := application.InvDataRepo.ListHistoricalMarketData(ctx, inst.ID, 10)
		if err != nil {
			t.Fatalf("failed to query bars after network failure: %v", err)
		}
		if len(afterBars) != initialCount {
			t.Errorf("existing data count changed! Before=%d, After=%d", initialCount, len(afterBars))
		}
	})
}
