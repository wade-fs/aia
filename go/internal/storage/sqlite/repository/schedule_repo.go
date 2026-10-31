package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/schedule"
	"github.com/wade/aia/internal/storage/sqlite"
)

type ScheduleRepository struct {
	db *sqlite.DB
}

func NewScheduleRepository(db *sqlite.DB) *ScheduleRepository {
	return &ScheduleRepository{db: db}
}

func (r *ScheduleRepository) Save(ctx context.Context, s *schedule.Schedule) error {
	var lr sql.NullString
	if s.LastRunAt != nil {
		lr = sql.NullString{String: s.LastRunAt.Format(time.RFC3339), Valid: true}
	}
	active := 0
	if s.IsActive {
		active = 1
	}

	_, err := r.db.RawDB().ExecContext(ctx, `
		INSERT INTO schedules (
			id, name, cron_expr, agent_id, task_template_json, misfire_policy,
			is_active, last_run_at, next_run_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			cron_expr = excluded.cron_expr,
			agent_id = excluded.agent_id,
			task_template_json = excluded.task_template_json,
			misfire_policy = excluded.misfire_policy,
			is_active = excluded.is_active,
			last_run_at = excluded.last_run_at,
			next_run_at = excluded.next_run_at,
			updated_at = excluded.updated_at
	`, s.ID, s.Name, s.CronExpr, s.AgentID, s.TaskTemplateJSON, string(s.MisfirePolicy),
		active, lr, s.NextRunAt.Format(time.RFC3339), s.CreatedAt.Format(time.RFC3339), s.UpdatedAt.Format(time.RFC3339))
	return err
}

func (r *ScheduleRepository) Get(ctx context.Context, id domain.ID) (*schedule.Schedule, error) {
	row := r.db.RawDB().QueryRowContext(ctx, `
		SELECT id, name, cron_expr, agent_id, task_template_json, misfire_policy,
			is_active, last_run_at, next_run_at, created_at, updated_at
		FROM schedules
		WHERE id = ? OR (length(?) >= 6 AND id LIKE ? || '%')
		LIMIT 1
	`, id, id, id)
	return scanSchedule(row)
}

func (r *ScheduleRepository) List(ctx context.Context, activeOnly bool) ([]schedule.Schedule, error) {
	query := `
		SELECT id, name, cron_expr, agent_id, task_template_json, misfire_policy,
			is_active, last_run_at, next_run_at, created_at, updated_at
		FROM schedules
	`
	if activeOnly {
		query += " WHERE is_active = 1"
	}
	query += " ORDER BY next_run_at ASC"

	rows, err := r.db.RawDB().QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []schedule.Schedule
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *s)
	}
	return list, rows.Err()
}

func (r *ScheduleRepository) ListDue(ctx context.Context, now time.Time) ([]schedule.Schedule, error) {
	rows, err := r.db.RawDB().QueryContext(ctx, `
		SELECT id, name, cron_expr, agent_id, task_template_json, misfire_policy,
			is_active, last_run_at, next_run_at, created_at, updated_at
		FROM schedules
		WHERE is_active = 1 AND next_run_at <= ?
		ORDER BY next_run_at ASC
	`, now.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []schedule.Schedule
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *s)
	}
	return list, rows.Err()
}

type scheduleScanner interface {
	Scan(dest ...any) error
}

func scanSchedule(s scheduleScanner) (*schedule.Schedule, error) {
	var sch schedule.Schedule
	var pol, lr, nr, ca, ua string
	var active int
	var lrNull sql.NullString
	if err := s.Scan(
		&sch.ID, &sch.Name, &sch.CronExpr, &sch.AgentID, &sch.TaskTemplateJSON, &pol,
		&active, &lrNull, &nr, &ca, &ua,
	); err != nil {
		return nil, err
	}
	sch.MisfirePolicy = schedule.MisfirePolicy(pol)
	sch.IsActive = (active == 1)
	if lrNull.Valid {
		t, _ := time.Parse(time.RFC3339, lrNull.String)
		sch.LastRunAt = &t
	}
	sch.NextRunAt, _ = time.Parse(time.RFC3339, nr)
	sch.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	sch.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
	_ = lr
	return &sch, nil
}
