package investment

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/wade/aia/internal/domain"
)

type MarketDataType string

const (
	MarketDataDailyBar  MarketDataType = "DAILY_BAR"
	MarketDataRealtime  MarketDataType = "REALTIME"
	MarketDataValuation MarketDataType = "VALUATION"
	MarketDataMacroRate MarketDataType = "MACRO_RATE"
)

// MarketBarValue represents parsed OHLCV and valuation metrics.
type MarketBarValue struct {
	Open          float64 `json:"open"`
	High          float64 `json:"high"`
	Low           float64 `json:"low"`
	Close         float64 `json:"close"`
	Volume        int64   `json:"volume"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
	PE            float64 `json:"pe,omitempty"`
	PB            float64 `json:"pb,omitempty"`
	DividendYield float64 `json:"dividend_yield,omitempty"`
}

// MarketData represents a price bar, valuation point, or macroeconomic index in SQLite (AIA-05 §18.2).
type MarketData struct {
	ID           domain.ID      `json:"id"`
	InstrumentID domain.ID      `json:"instrument_id"`
	DataType     MarketDataType `json:"data_type"`
	Period       string         `json:"period"` // "1D", "5M", "1M"
	ValueJSON    string         `json:"value_json"`
	SourceID     domain.ID      `json:"source_id"`
	FetchedAt    time.Time      `json:"fetched_at"`
	EffectiveAt  time.Time      `json:"effective_at"`
}

// Validate ensures temporal and pricing sanity.
func (m *MarketData) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("market data id cannot be empty")
	}
	if m.InstrumentID == "" {
		return fmt.Errorf("market data instrument_id cannot be empty")
	}
	if m.SourceID == "" {
		return fmt.Errorf("market data source_id cannot be empty")
	}
	if m.EffectiveAt.IsZero() {
		return fmt.Errorf("effective_at must be provided")
	}
	// Effective time cannot be far into the future (allow 10m clock skew)
	if m.EffectiveAt.After(time.Now().Add(10 * time.Minute)) {
		return fmt.Errorf("effective_at %v is in the future", m.EffectiveAt)
	}

	// Verify value_json syntax and basic numerical bounds
	var bar MarketBarValue
	if err := json.Unmarshal([]byte(m.ValueJSON), &bar); err != nil {
		return fmt.Errorf("invalid value_json payload: %w", err)
	}
	if bar.Close <= 0 && m.DataType == MarketDataDailyBar {
		return fmt.Errorf("daily bar closing price must be > 0 (got %f)", bar.Close)
	}
	if bar.Volume < 0 {
		return fmt.Errorf("volume cannot be negative (got %d)", bar.Volume)
	}

	if m.FetchedAt.IsZero() {
		m.FetchedAt = time.Now().UTC()
	}
	return nil
}
