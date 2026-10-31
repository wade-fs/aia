package unit

import (
	"testing"
	"time"

	"github.com/wade/aia/internal/domain/schedule"
)

func TestComputeNextRunAliases(t *testing.T) {
	now := time.Date(2026, 10, 31, 14, 30, 0, 0, time.UTC)

	// @hourly
	nextHourly, err := schedule.ComputeNextRun("@hourly", now)
	if err != nil {
		t.Fatalf("unexpected error for @hourly: %v", err)
	}
	expectedHourly := time.Date(2026, 10, 31, 15, 0, 0, 0, time.UTC)
	if !nextHourly.Equal(expectedHourly) {
		t.Errorf("expected %v, got %v", expectedHourly, nextHourly)
	}

	// @daily
	nextDaily, err := schedule.ComputeNextRun("@daily", now)
	if err != nil {
		t.Fatalf("unexpected error for @daily: %v", err)
	}
	expectedDaily := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	if !nextDaily.Equal(expectedDaily) {
		t.Errorf("expected %v, got %v", expectedDaily, nextDaily)
	}
}

func TestComputeNextRunCronExpression(t *testing.T) {
	now := time.Date(2026, 10, 31, 14, 25, 0, 0, time.UTC)

	// Every 15 minutes: */15 * * * *
	next15, err := schedule.ComputeNextRun("*/15 * * * *", now)
	if err != nil {
		t.Fatalf("unexpected error for */15 * * * *: %v", err)
	}
	expected15 := time.Date(2026, 10, 31, 14, 30, 0, 0, time.UTC)
	if !next15.Equal(expected15) {
		t.Errorf("expected %v, got %v", expected15, next15)
	}

	// Specific time: 30 9 * * 1-5 (Weekday morning 09:30)
	// 2026-10-31 is a Saturday, so next weekday is Monday 2026-11-02
	nextMarket, err := schedule.ComputeNextRun("30 9 * * 1-5", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nextMarket.Weekday() != time.Monday {
		t.Errorf("expected Monday, got %v", nextMarket.Weekday())
	}
	if nextMarket.Hour() != 9 || nextMarket.Minute() != 30 {
		t.Errorf("expected 09:30, got %02d:%02d", nextMarket.Hour(), nextMarket.Minute())
	}
}
