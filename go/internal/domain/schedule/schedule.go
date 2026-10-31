package schedule

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/wade/aia/internal/domain"
)

type MisfirePolicy string

const (
	MisfireSkip    MisfirePolicy = "SKIP"
	MisfireRunOnce MisfirePolicy = "RUN_ONCE"
	MisfireCatchUp MisfirePolicy = "CATCH_UP"
)

// Schedule represents a periodic automated task trigger.
type Schedule struct {
	ID               domain.ID     `json:"id"`
	Name             string        `json:"name"`
	CronExpr         string        `json:"cron_expr"`
	AgentID          domain.ID     `json:"agent_id"`
	TaskTemplateJSON string        `json:"task_template_json"`
	MisfirePolicy    MisfirePolicy `json:"misfire_policy"`
	IsActive         bool          `json:"is_active"`
	LastRunAt        *time.Time    `json:"last_run_at,omitempty"`
	NextRunAt        time.Time     `json:"next_run_at"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// ComputeNextRun calculates the next occurrence after 'after'.
// Supports standard 5-part cron syntax (minute, hour, day of month, month, day of week)
// or standard aliases like @daily, @hourly, @weekly.
func ComputeNextRun(expr string, after time.Time) (time.Time, error) {
	expr = strings.TrimSpace(expr)
	switch expr {
	case "@hourly":
		return after.Truncate(time.Hour).Add(time.Hour), nil
	case "@daily":
		next := time.Date(after.Year(), after.Month(), after.Day(), 0, 0, 0, 0, after.Location()).AddDate(0, 0, 1)
		return next, nil
	case "@weekly":
		daysUntilNextSunday := (7 - int(after.Weekday())) % 7
		if daysUntilNextSunday == 0 {
			daysUntilNextSunday = 7
		}
		next := time.Date(after.Year(), after.Month(), after.Day(), 0, 0, 0, 0, after.Location()).AddDate(0, 0, daysUntilNextSunday)
		return next, nil
	}

	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return time.Time{}, fmt.Errorf("invalid cron expression %q: expected 5 fields or alias", expr)
	}

	// Step forward by 1 minute increments until match, up to 1 year limit
	curr := after.Truncate(time.Minute).Add(time.Minute)
	limit := after.AddDate(1, 0, 0)

	for curr.Before(limit) {
		if matchField(parts[0], curr.Minute(), 0, 59) &&
			matchField(parts[1], curr.Hour(), 0, 23) &&
			matchField(parts[2], curr.Day(), 1, 31) &&
			matchField(parts[3], int(curr.Month()), 1, 12) &&
			matchField(parts[4], int(curr.Weekday()), 0, 6) {
			return curr, nil
		}
		curr = curr.Add(time.Minute)
	}

	return time.Time{}, fmt.Errorf("could not find next run time within 1 year for expr %q", expr)
}

func matchField(field string, val int, min, max int) bool {
	if field == "*" {
		return true
	}
	// Step: */5
	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(strings.TrimPrefix(field, "*/"))
		if err == nil && step > 0 {
			return val%step == 0
		}
	}
	// Comma separated list: 1,2,3
	if strings.Contains(field, ",") {
		subfields := strings.Split(field, ",")
		for _, s := range subfields {
			if matchField(strings.TrimSpace(s), val, min, max) {
				return true
			}
		}
		return false
	}
	// Range: 1-5
	if strings.Contains(field, "-") {
		parts := strings.Split(field, "-")
		if len(parts) == 2 {
			start, err1 := strconv.Atoi(parts[0])
			end, err2 := strconv.Atoi(parts[1])
			if err1 == nil && err2 == nil {
				return val >= start && val <= end
			}
		}
	}
	// Exact value
	num, err := strconv.Atoi(field)
	if err == nil {
		return num == val
	}
	return false
}
