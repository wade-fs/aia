package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/wade/aia/internal/autonomy/clock"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/investment"
	"github.com/wade/aia/internal/storage/sqlite"
)

// InvestmentDataRepository manages persistence for DataSources, RawDocuments (with deduplication),
// MarketData, and immutable Research DataSnapshots (AIA-05 §18.1, §18.2).
type InvestmentDataRepository struct {
	db *sqlite.DB
}

func NewInvestmentDataRepository(db *sqlite.DB) *InvestmentDataRepository {
	return &InvestmentDataRepository{db: db}
}

// ----------------------------------------------------------------
// 1. Data Sources
// ----------------------------------------------------------------

func (r *InvestmentDataRepository) SaveDataSource(ctx context.Context, s *investment.DataSource) error {
	if err := s.Validate(); err != nil {
		return err
	}
	now := time.Now().UTC()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now

	var ls, lf sql.NullString
	if s.LastSuccessAt != nil {
		ls = sql.NullString{String: s.LastSuccessAt.Format(time.RFC3339), Valid: true}
	}
	if s.LastFailureAt != nil {
		lf = sql.NullString{String: s.LastFailureAt.Format(time.RFC3339), Valid: true}
	}

	_, err := r.db.RawDB().ExecContext(ctx, `
		INSERT INTO investment_data_sources (
			id, source_name, source_type, base_url, documentation_url,
			authentication_type, configuration_json, status, health_status,
			last_success_at, last_failure_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			source_name = excluded.source_name,
			source_type = excluded.source_type,
			base_url = excluded.base_url,
			documentation_url = excluded.documentation_url,
			authentication_type = excluded.authentication_type,
			configuration_json = excluded.configuration_json,
			status = excluded.status,
			health_status = excluded.health_status,
			last_success_at = excluded.last_success_at,
			last_failure_at = excluded.last_failure_at,
			updated_at = excluded.updated_at
	`, s.ID, s.SourceName, s.SourceType, s.BaseURL, s.DocumentationURL,
		s.AuthenticationType, s.ConfigurationJSON, string(s.Status), string(s.HealthStatus),
		ls, lf, s.CreatedAt.Format(time.RFC3339), s.UpdatedAt.Format(time.RFC3339))
	return err
}

func (r *InvestmentDataRepository) GetDataSource(ctx context.Context, id domain.ID) (*investment.DataSource, error) {
	row := r.db.RawDB().QueryRowContext(ctx, `
		SELECT id, source_name, source_type, base_url, documentation_url, authentication_type,
			configuration_json, status, health_status, last_success_at, last_failure_at, created_at, updated_at
		FROM investment_data_sources WHERE id = ?
	`, id)
	return scanDataSource(row)
}

func (r *InvestmentDataRepository) GetDataSourceByName(ctx context.Context, name string) (*investment.DataSource, error) {
	row := r.db.RawDB().QueryRowContext(ctx, `
		SELECT id, source_name, source_type, base_url, documentation_url, authentication_type,
			configuration_json, status, health_status, last_success_at, last_failure_at, created_at, updated_at
		FROM investment_data_sources WHERE source_name = ?
	`, name)
	return scanDataSource(row)
}

func (r *InvestmentDataRepository) ListDataSources(ctx context.Context) ([]investment.DataSource, error) {
	rows, err := r.db.RawDB().QueryContext(ctx, `
		SELECT id, source_name, source_type, base_url, documentation_url, authentication_type,
			configuration_json, status, health_status, last_success_at, last_failure_at, created_at, updated_at
		FROM investment_data_sources ORDER BY source_name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []investment.DataSource
	for rows.Next() {
		s, err := scanDataSource(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *s)
	}
	return list, rows.Err()
}

func (r *InvestmentDataRepository) UpdateDataSourceHealth(ctx context.Context, id domain.ID, health investment.SourceHealth, isSuccess bool) error {
	now := time.Now().UTC().Format(time.RFC3339)
	var query string
	if isSuccess {
		query = "UPDATE investment_data_sources SET health_status = ?, last_success_at = ?, updated_at = ? WHERE id = ?"
	} else {
		query = "UPDATE investment_data_sources SET health_status = ?, last_failure_at = ?, updated_at = ? WHERE id = ?"
	}
	_, err := r.db.RawDB().ExecContext(ctx, query, string(health), now, now, id)
	return err
}

// ----------------------------------------------------------------
// 2. Raw Documents & Deduplication
// ----------------------------------------------------------------

// SaveRawDocument persists a document and detects duplicates by content_hash or source_record_id.
// Returns (isDuplicate, error). If duplicate, the database record is preserved without error.
func (r *InvestmentDataRepository) SaveRawDocument(ctx context.Context, doc *investment.RawDocument) (bool, error) {
	if err := doc.Validate(); err != nil {
		return false, err
	}

	// 1. Check duplicate by (source_id, content_hash)
	var existingID string
	err := r.db.RawDB().QueryRowContext(ctx,
		"SELECT id FROM investment_raw_documents WHERE source_id = ? AND content_hash = ? LIMIT 1",
		doc.SourceID, doc.ContentHash).Scan(&existingID)
	if err == nil && existingID != "" {
		doc.ID = existingID
		return true, nil // duplicate identified and skipped
	}

	// 2. Insert new document
	_, err = r.db.RawDB().ExecContext(ctx, `
		INSERT INTO investment_raw_documents (
			id, source_id, instrument_id, source_record_id, source_url, title, summary,
			content_reference, content_hash, content_type, published_at, fetched_at, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, doc.ID, doc.SourceID, doc.InstrumentID, doc.SourceRecordID, doc.SourceURL, doc.Title, doc.Summary,
		doc.ContentReference, doc.ContentHash, string(doc.ContentType), doc.PublishedAt.Format(time.RFC3339),
		doc.FetchedAt.Format(time.RFC3339), doc.MetadataJSON)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (r *InvestmentDataRepository) GetRawDocumentByHash(ctx context.Context, sourceID domain.ID, hash string) (*investment.RawDocument, error) {
	row := r.db.RawDB().QueryRowContext(ctx, `
		SELECT id, source_id, instrument_id, source_record_id, source_url, title, summary,
			content_reference, content_hash, content_type, published_at, fetched_at, metadata_json
		FROM investment_raw_documents WHERE source_id = ? AND content_hash = ? LIMIT 1
	`, sourceID, hash)
	return scanRawDocument(row)
}

func (r *InvestmentDataRepository) ListRawDocuments(ctx context.Context, instrumentID domain.ID, limit int) ([]investment.RawDocument, error) {
	if limit <= 0 {
		limit = 30
	}
	query := `
		SELECT id, source_id, instrument_id, source_record_id, source_url, title, summary,
			content_reference, content_hash, content_type, published_at, fetched_at, metadata_json
		FROM investment_raw_documents
		WHERE 1 = 1`
	var args []any
	if instrumentID != "" {
		query += ` AND instrument_id = ?`
		args = append(args, instrumentID)
	}
	if asOf, ok := clock.AsOf(ctx); ok {
		query += ` AND published_at <= ?`
		args = append(args, asOf.Format(time.RFC3339))
	}
	query += ` ORDER BY published_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := r.db.RawDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []investment.RawDocument
	for rows.Next() {
		doc, err := scanRawDocument(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *doc)
	}
	return list, rows.Err()
}

// ----------------------------------------------------------------
// 3. Market Data
// ----------------------------------------------------------------

func (r *InvestmentDataRepository) SaveMarketData(ctx context.Context, md *investment.MarketData) error {
	if err := md.Validate(); err != nil {
		return err
	}
	_, err := r.db.RawDB().ExecContext(ctx, `
		INSERT INTO investment_market_data (
			id, instrument_id, data_type, period, value_json, source_id, fetched_at, effective_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, md.ID, md.InstrumentID, string(md.DataType), md.Period, md.ValueJSON, md.SourceID,
		md.FetchedAt.Format(time.RFC3339), md.EffectiveAt.Format(time.RFC3339))
	return err
}

func (r *InvestmentDataRepository) GetLatestMarketData(ctx context.Context, instrumentID domain.ID) (*investment.MarketData, error) {
	query := `
		SELECT id, instrument_id, data_type, period, value_json, source_id, fetched_at, effective_at
		FROM investment_market_data
		WHERE instrument_id = ?`
	args := []any{instrumentID}
	if asOf, ok := clock.AsOf(ctx); ok {
		query += ` AND effective_at <= ?`
		args = append(args, asOf.Format(time.RFC3339))
	}
	query += ` ORDER BY effective_at DESC, rowid DESC LIMIT 1`
	row := r.db.RawDB().QueryRowContext(ctx, query, args...)
	return scanMarketData(row)
}

func (r *InvestmentDataRepository) ListHistoricalMarketData(ctx context.Context, instrumentID domain.ID, limit int) ([]investment.MarketData, error) {
	if limit <= 0 {
		limit = 30
	}
	query := `
		SELECT id, instrument_id, data_type, period, value_json, source_id, fetched_at, effective_at
		FROM investment_market_data
		WHERE instrument_id = ?`
	args := []any{instrumentID}
	if asOf, ok := clock.AsOf(ctx); ok {
		query += ` AND effective_at <= ?`
		args = append(args, asOf.Format(time.RFC3339))
	}
	query += ` ORDER BY effective_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := r.db.RawDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []investment.MarketData
	for rows.Next() {
		md, err := scanMarketData(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *md)
	}
	return list, rows.Err()
}

// ----------------------------------------------------------------
// 4. Research Data Snapshots & Items (AIA-05 §10)
// ----------------------------------------------------------------

func (r *InvestmentDataRepository) CreateDataSnapshot(ctx context.Context, snap *investment.DataSnapshot, items []investment.SnapshotItem) error {
	if err := snap.Validate(); err != nil {
		return err
	}
	now := time.Now().UTC()
	if snap.CreatedAt.IsZero() {
		snap.CreatedAt = now
	}

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO investment_data_snapshots (
				id, snapshot_type, data_cutoff_at, instrument_scope_json, source_versions_json,
				validation_summary_json, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)
		`, snap.ID, string(snap.SnapshotType), snap.DataCutoffAt.Format(time.RFC3339),
			snap.InstrumentScopeJSON, snap.SourceVersionsJSON, snap.ValidationSummaryJSON,
			snap.CreatedAt.Format(time.RFC3339))
		if err != nil {
			return err
		}

		for _, it := range items {
			if it.CreatedAt.IsZero() {
				it.CreatedAt = now
			}
			_, err = tx.ExecContext(ctx, `
				INSERT INTO investment_snapshot_items (
					id, snapshot_id, item_type, reference_id, payload_json, created_at
				) VALUES (?, ?, ?, ?, ?, ?)
			`, it.ID, snap.ID, string(it.ItemType), it.ReferenceID, it.PayloadJSON, it.CreatedAt.Format(time.RFC3339))
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *InvestmentDataRepository) GetDataSnapshot(ctx context.Context, id domain.ID) (*investment.DataSnapshot, error) {
	row := r.db.RawDB().QueryRowContext(ctx, `
		SELECT id, snapshot_type, data_cutoff_at, instrument_scope_json, source_versions_json,
			validation_summary_json, created_at
		FROM investment_data_snapshots WHERE id = ?
	`, id)
	return scanDataSnapshot(row)
}

func (r *InvestmentDataRepository) ListSnapshotItems(ctx context.Context, snapshotID domain.ID) ([]investment.SnapshotItem, error) {
	rows, err := r.db.RawDB().QueryContext(ctx, `
		SELECT id, snapshot_id, item_type, reference_id, payload_json, created_at
		FROM investment_snapshot_items WHERE snapshot_id = ?
		ORDER BY created_at ASC
	`, snapshotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []investment.SnapshotItem
	for rows.Next() {
		var it investment.SnapshotItem
		var ity, ca string
		if err := rows.Scan(&it.ID, &it.SnapshotID, &ity, &it.ReferenceID, &it.PayloadJSON, &ca); err != nil {
			return nil, err
		}
		it.ItemType = investment.SnapshotItemType(ity)
		it.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		list = append(list, it)
	}
	return list, rows.Err()
}

// ----------------------------------------------------------------
// Scanners
// ----------------------------------------------------------------

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDataSource(s rowScanner) (*investment.DataSource, error) {
	var d investment.DataSource
	var st, hs, ls, lf, ca, ua string
	var lsNull, lfNull sql.NullString
	if err := s.Scan(
		&d.ID, &d.SourceName, &d.SourceType, &d.BaseURL, &d.DocumentationURL, &d.AuthenticationType,
		&d.ConfigurationJSON, &st, &hs, &lsNull, &lfNull, &ca, &ua,
	); err != nil {
		return nil, err
	}
	d.Status = investment.SourceStatus(st)
	d.HealthStatus = investment.SourceHealth(hs)
	if lsNull.Valid {
		t, _ := time.Parse(time.RFC3339, lsNull.String)
		d.LastSuccessAt = &t
	}
	if lfNull.Valid {
		t, _ := time.Parse(time.RFC3339, lfNull.String)
		d.LastFailureAt = &t
	}
	d.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	d.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
	_ = ls
	_ = lf
	return &d, nil
}

func scanRawDocument(s rowScanner) (*investment.RawDocument, error) {
	var doc investment.RawDocument
	var ct, pa, fa string
	if err := s.Scan(
		&doc.ID, &doc.SourceID, &doc.InstrumentID, &doc.SourceRecordID, &doc.SourceURL,
		&doc.Title, &doc.Summary, &doc.ContentReference, &doc.ContentHash, &ct,
		&pa, &fa, &doc.MetadataJSON,
	); err != nil {
		return nil, err
	}
	doc.ContentType = investment.DocumentContentType(ct)
	doc.PublishedAt, _ = time.Parse(time.RFC3339, pa)
	doc.FetchedAt, _ = time.Parse(time.RFC3339, fa)
	return &doc, nil
}

func scanMarketData(s rowScanner) (*investment.MarketData, error) {
	var md investment.MarketData
	var dt, fa, ea string
	if err := s.Scan(&md.ID, &md.InstrumentID, &dt, &md.Period, &md.ValueJSON, &md.SourceID, &fa, &ea); err != nil {
		return nil, err
	}
	md.DataType = investment.MarketDataType(dt)
	md.FetchedAt, _ = time.Parse(time.RFC3339, fa)
	md.EffectiveAt, _ = time.Parse(time.RFC3339, ea)
	return &md, nil
}

func scanDataSnapshot(s rowScanner) (*investment.DataSnapshot, error) {
	var snap investment.DataSnapshot
	var st, dc, ca string
	if err := s.Scan(
		&snap.ID, &st, &dc, &snap.InstrumentScopeJSON, &snap.SourceVersionsJSON,
		&snap.ValidationSummaryJSON, &ca,
	); err != nil {
		return nil, err
	}
	snap.SnapshotType = investment.SnapshotType(st)
	snap.DataCutoffAt, _ = time.Parse(time.RFC3339, dc)
	snap.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	return &snap, nil
}
