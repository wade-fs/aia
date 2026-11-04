package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/wade/aia/internal/domain/schedule"
	taskdomain "github.com/wade/aia/internal/service/task"
)

type ScheduleWorker struct {
	mgr *Manager
}

func newScheduleWorker(mgr *Manager) *ScheduleWorker {
	return &ScheduleWorker{mgr: mgr}
}

// Run executes the periodic schedule scanning loop.
func (w *ScheduleWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.mgr.cfg.ScheduleInterval)
	defer ticker.Stop()

	slog.Info("schedule worker loop started", "interval", w.mgr.cfg.ScheduleInterval)

	for {
		select {
		case <-ctx.Done():
			slog.Info("schedule worker received stop signal")
			return
		case <-ticker.C:
			if !w.mgr.cfg.AutoDispatchEnabled {
				continue
			}
			dispatched, err := w.mgr.DispatchDue(ctx)
			if err != nil {
				slog.Error("error processing due schedules", "error", err)
			} else if dispatched > 0 {
				slog.Info("dispatched due scheduled tasks", "count", dispatched)
			}
		}
	}
}

// SeedDefaultSchedules ensures standard automation schedules for the three frequencies exist.
func (m *Manager) SeedDefaultSchedules(ctx context.Context) error {
	if m.scheduleSvc == nil {
		return nil
	}

	existing, err := m.scheduleSvc.ListSchedules(ctx, false)
	if err != nil {
		return err
	}

	existsByName := make(map[string]bool)
	for _, s := range existing {
		existsByName[strings.ToLower(strings.TrimSpace(s.Name))] = true
	}

	defaultSchedules := []struct {
		Name          string
		CronExpr      string
		AgentID       string
		MisfirePolicy schedule.MisfirePolicy
		Template      taskdomain.CreateTaskRequest
	}{
		// 1. Low Frequency: TWSE Daily Market Close Sync (14:30 weekdays)
		{
			Name:          "TWSE Daily Market Close Sync",
			CronExpr:      "30 14 * * 1-5",
			AgentID:       "system-collector",
			MisfirePolicy: schedule.MisfireSkip,
			Template: taskdomain.CreateTaskRequest{
				Title:       "TWSE Daily Market Close Sync",
				Description: "Sync official TWSE/TPEx daily closing quotes and trading volumes for watchlist instruments.",
				Acceptance:  []string{"twse_sync", "data_collection"},
				Priority:    40,
			},
		},
		// 2. Low Frequency: MOPS Announcements Check (every 4 hours)
		{
			Name:          "MOPS Announcements Check",
			CronExpr:      "0 */4 * * *",
			AgentID:       "system-collector",
			MisfirePolicy: schedule.MisfireSkip,
			Template: taskdomain.CreateTaskRequest{
				Title:       "MOPS Announcements Check",
				Description: "Collect MOPS corporate announcements, material filings, and semiconductor industry news.",
				Acceptance:  []string{"mops_sync", "data_collection"},
				Priority:    45,
			},
		},
		// 3. Low Frequency: FRED Macro Indicators Sync (every Monday 08:00)
		{
			Name:          "FRED Macro Indicators Sync",
			CronExpr:      "0 8 * * 1",
			AgentID:       "system-collector",
			MisfirePolicy: schedule.MisfireSkip,
			Template: taskdomain.CreateTaskRequest{
				Title:       "FRED Macro Indicators Sync",
				Description: "Update US 10-year Treasury yield, Fed Funds rate, and macro indicators from FRED.",
				Acceptance:  []string{"fred_sync", "data_collection"},
				Priority:    35,
			},
		},
		// 4. Low Frequency: Prediction Verification and Calibration (every Friday 18:00)
		{
			Name:          "Prediction Verification and Calibration",
			CronExpr:      "0 18 * * 5",
			AgentID:       "system-evaluator",
			MisfirePolicy: schedule.MisfireSkip,
			Template: taskdomain.CreateTaskRequest{
				Title:       "Prediction Verification and Calibration",
				Description: "Evaluate due forecasts against actual settlement prices, calculate direction accuracy and Brier score calibration.",
				Acceptance:  []string{"prediction_eval"},
				Priority:    30,
			},
		},
		// 5. Medium Frequency: Investment Research Gated Evaluation (weekdays 16:00, Materiality Gated)
		{
			Name:          "Investment Research Gated Evaluation",
			CronExpr:      "0 16 * * 1-5",
			AgentID:       "investment-agent",
			MisfirePolicy: schedule.MisfireSkip,
			Template: taskdomain.CreateTaskRequest{
				Title:       "Investment Research Gated Evaluation",
				Description: "Evaluate semiconductor portfolio and watchlist. Gated by materiality: only update thesis/proposals when material changes occur.",
				Acceptance:  []string{"materiality_gated_research"},
				Priority:    50,
			},
		},
		// 6. Maintenance: Daily Log Retention and Rotation Purge (02:00 daily, AIA-19)
		{
			Name:          "Daily Log Retention Purge",
			CronExpr:      "0 2 * * *",
			AgentID:       "system-operator",
			MisfirePolicy: schedule.MisfireSkip,
			Template: taskdomain.CreateTaskRequest{
				Title:       "Daily Log Retention and Expired Log Purge",
				Description: "Enforce logging.retain_days policy by purging obsolete audit logs and execution event history.",
				Acceptance:  []string{"log_retention", "maintenance"},
				Priority:    10,
			},
		},
	}

	for _, ds := range defaultSchedules {
		normName := strings.ToLower(strings.TrimSpace(ds.Name))
		if existsByName[normName] {
			continue
		}

		tplJSON, _ := json.Marshal(ds.Template)
		_, err := m.scheduleSvc.CreateSchedule(
			ctx,
			ds.Name,
			ds.CronExpr,
			ds.AgentID,
			string(tplJSON),
			ds.MisfirePolicy,
		)
		if err != nil {
			slog.Warn("failed to create default schedule", "name", ds.Name, "error", err)
		} else {
			slog.Info("seeded default recurring schedule", "name", ds.Name, "cron", ds.CronExpr)
		}
	}

	return nil
}
