package worker

import (
	"context"
	"log/slog"
	"strings"
	"time"
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

// SeedDefaultSchedules cleans up any legacy default schedules.
// Per system requirement, all preset default schedules/tasks have been deleted and are no longer seeded.
func (m *Manager) SeedDefaultSchedules(ctx context.Context) error {
	if m.scheduleSvc == nil {
		return nil
	}

	defaultNames := []string{
		"twse daily market close sync",
		"mops announcements check",
		"fred macro indicators sync",
		"prediction verification and calibration",
		"investment research gated evaluation",
		"daily log retention purge",
	}

	existing, err := m.scheduleSvc.ListSchedules(ctx, false)
	if err != nil {
		return err
	}

	for _, s := range existing {
		normName := strings.ToLower(strings.TrimSpace(s.Name))
		for _, dName := range defaultNames {
			if normName == dName {
				_ = m.scheduleSvc.DeleteSchedule(ctx, s.ID)
				slog.Info("purged legacy default schedule", "name", s.Name, "id", s.ID)
				break
			}
		}
	}

	return nil
}
