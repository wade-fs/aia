package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wade/aia/internal/autonomy/clock"
	domainExp "github.com/wade/aia/internal/domain/experiment"
	"github.com/wade/aia/internal/storage/sqlite"
)

type ExperimentRepository struct {
	db *sqlite.DB
}

func NewExperimentRepository(db *sqlite.DB) *ExperimentRepository {
	return &ExperimentRepository{db: db}
}

func (r *ExperimentRepository) CreateExperiment(ctx context.Context, exp *domainExp.Experiment) error {
	now := clock.Now(ctx)
	if exp.CreatedAt.IsZero() {
		exp.CreatedAt = now
	}
	if exp.UpdatedAt.IsZero() {
		exp.UpdatedAt = exp.CreatedAt
	}
	if exp.BaselineMetricsJSON == "" {
		exp.BaselineMetricsJSON = "{}"
	}
	if exp.CurrentMetricsJSON == "" {
		exp.CurrentMetricsJSON = "{}"
	}

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO experiments (
				id, project_id, primary_goal_id, domain, owner_agent_id, name, hypothesis,
				experiment_type, status, baseline_metrics_json, current_metrics_json,
				iteration_count, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, exp.ID, exp.ProjectID, exp.PrimaryGoalID, exp.Domain, exp.OwnerAgentID, exp.Name,
			exp.Hypothesis, exp.ExperimentType, string(exp.Status), exp.BaselineMetricsJSON,
			exp.CurrentMetricsJSON, exp.IterationCount,
			exp.CreatedAt.Format(time.RFC3339), exp.UpdatedAt.Format(time.RFC3339))
		if err != nil {
			return err
		}
		if exp.PrimaryGoalID != nil && *exp.PrimaryGoalID != "" {
			_, err = tx.ExecContext(ctx, `
				INSERT OR IGNORE INTO experiment_goal_links (experiment_id, goal_id, role, created_at)
				VALUES (?, ?, 'PRIMARY', ?)
			`, exp.ID, *exp.PrimaryGoalID, exp.CreatedAt.Format(time.RFC3339))
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *ExperimentRepository) GetExperiment(ctx context.Context, id string) (*domainExp.Experiment, error) {
	row := r.db.RawDB().QueryRowContext(ctx, `
		SELECT id, project_id, primary_goal_id, domain, owner_agent_id, name, hypothesis,
			experiment_type, status, baseline_metrics_json, current_metrics_json,
			iteration_count, created_at, updated_at
		FROM experiments WHERE id = ?
	`, id)
	return scanExperiment(row)
}

func (r *ExperimentRepository) ListExperiments(ctx context.Context, domainFilter string, limit int) ([]domainExp.Experiment, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, project_id, primary_goal_id, domain, owner_agent_id, name, hypothesis,
			experiment_type, status, baseline_metrics_json, current_metrics_json,
			iteration_count, created_at, updated_at
		FROM experiments WHERE 1=1`
	var args []any
	if domainFilter != "" {
		query += ` AND domain = ?`
		args = append(args, domainFilter)
	}
	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.RawDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domainExp.Experiment
	for rows.Next() {
		exp, err := scanExperiment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *exp)
	}
	return out, rows.Err()
}

func (r *ExperimentRepository) IncrementIterationAndMetrics(ctx context.Context, experimentID string, currentMetricsJSON string) error {
	now := clock.Now(ctx)
	_, err := r.db.RawDB().ExecContext(ctx, `
		UPDATE experiments
		SET iteration_count = iteration_count + 1,
		    current_metrics_json = ?,
		    updated_at = ?
		WHERE id = ?
	`, currentMetricsJSON, now.Format(time.RFC3339), experimentID)
	return err
}

func scanExperiment(scanner interface{ Scan(dest ...any) error }) (*domainExp.Experiment, error) {
	var e domainExp.Experiment
	var projID, goalID sql.NullString
	var st, ca, ua string
	err := scanner.Scan(
		&e.ID, &projID, &goalID, &e.Domain, &e.OwnerAgentID, &e.Name, &e.Hypothesis,
		&e.ExperimentType, &st, &e.BaselineMetricsJSON, &e.CurrentMetricsJSON,
		&e.IterationCount, &ca, &ua,
	)
	if err != nil {
		return nil, err
	}
	if projID.Valid {
		v := projID.String
		e.ProjectID = &v
	}
	if goalID.Valid {
		v := goalID.String
		e.PrimaryGoalID = &v
	}
	e.Status = domainExp.Status(st)
	e.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	e.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
	return &e, nil
}

func (r *ExperimentRepository) CreateGeneration(ctx context.Context, gen *domainExp.ExperimentGeneration) error {
	now := clock.Now(ctx)
	if gen.CreatedAt.IsZero() {
		gen.CreatedAt = now
	}
	if gen.UpdatedAt.IsZero() {
		gen.UpdatedAt = gen.CreatedAt
	}
	if gen.StrategyConfigJSON == "" {
		gen.StrategyConfigJSON = "{}"
	}
	frozenInt := 0
	if gen.IsFrozen {
		frozenInt = 1
	}

	_, err := r.db.RawDB().ExecContext(ctx, `
		INSERT INTO experiment_generations (
			id, experiment_id, generation_code, parent_generation_id, role, engine_type,
			portfolio_profile_id, is_frozen, harness_version, model_id, prompt_version,
			strategy_version, strategy_config_json, source_correction_id, source_learning_id,
			status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, gen.ID, gen.ExperimentID, gen.GenerationCode, gen.ParentGenerationID,
		string(gen.Role), string(gen.EngineType), gen.PortfolioProfileID, frozenInt,
		gen.HarnessVersion, gen.ModelID, gen.PromptVersion, gen.StrategyVersion,
		gen.StrategyConfigJSON, gen.SourceCorrectionID, gen.SourceLearningID,
		string(gen.Status), gen.CreatedAt.Format(time.RFC3339), gen.UpdatedAt.Format(time.RFC3339))
	return err
}

func (r *ExperimentRepository) GetGeneration(ctx context.Context, id string) (*domainExp.ExperimentGeneration, error) {
	row := r.db.RawDB().QueryRowContext(ctx, `
		SELECT id, experiment_id, generation_code, parent_generation_id, role, engine_type,
			portfolio_profile_id, is_frozen, harness_version, model_id, prompt_version,
			strategy_version, strategy_config_json, source_correction_id, source_learning_id,
			status, created_at, updated_at
		FROM experiment_generations WHERE id = ?
	`, id)
	return scanGeneration(row)
}

func (r *ExperimentRepository) GetGenerationByCode(ctx context.Context, experimentID string, code string) (*domainExp.ExperimentGeneration, error) {
	row := r.db.RawDB().QueryRowContext(ctx, `
		SELECT id, experiment_id, generation_code, parent_generation_id, role, engine_type,
			portfolio_profile_id, is_frozen, harness_version, model_id, prompt_version,
			strategy_version, strategy_config_json, source_correction_id, source_learning_id,
			status, created_at, updated_at
		FROM experiment_generations WHERE experiment_id = ? AND generation_code = ?
	`, experimentID, code)
	return scanGeneration(row)
}

func (r *ExperimentRepository) ListGenerations(ctx context.Context, experimentID string) ([]domainExp.ExperimentGeneration, error) {
	query := `
		SELECT id, experiment_id, generation_code, parent_generation_id, role, engine_type,
			portfolio_profile_id, is_frozen, harness_version, model_id, prompt_version,
			strategy_version, strategy_config_json, source_correction_id, source_learning_id,
			status, created_at, updated_at
		FROM experiment_generations WHERE experiment_id = ?`
	args := []any{experimentID}
	if asOf, ok := clock.AsOf(ctx); ok {
		query += ` AND created_at <= ?`
		args = append(args, asOf.Format(time.RFC3339))
	}
	query += ` ORDER BY created_at ASC, rowid ASC`

	rows, err := r.db.RawDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domainExp.ExperimentGeneration
	for rows.Next() {
		g, err := scanGeneration(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *g)
	}
	return out, rows.Err()
}

func (r *ExperimentRepository) UpdateGenerationStatus(ctx context.Context, generationID string, status domainExp.GenerationStatus) error {
	now := clock.Now(ctx)
	res, err := r.db.RawDB().ExecContext(ctx, `
		UPDATE experiment_generations
		SET status = ?, updated_at = ?
		WHERE id = ?
	`, string(status), now.Format(time.RFC3339), generationID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("generation %s not found", generationID)
	}
	return nil
}

func scanGeneration(scanner interface{ Scan(dest ...any) error }) (*domainExp.ExperimentGeneration, error) {
	var g domainExp.ExperimentGeneration
	var parentID, corrID, learnID sql.NullString
	var role, eng, st, ca, ua string
	var frozenInt int
	err := scanner.Scan(
		&g.ID, &g.ExperimentID, &g.GenerationCode, &parentID, &role, &eng,
		&g.PortfolioProfileID, &frozenInt, &g.HarnessVersion, &g.ModelID, &g.PromptVersion,
		&g.StrategyVersion, &g.StrategyConfigJSON, &corrID, &learnID,
		&st, &ca, &ua,
	)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		v := parentID.String
		g.ParentGenerationID = &v
	}
	if corrID.Valid {
		v := corrID.String
		g.SourceCorrectionID = &v
	}
	if learnID.Valid {
		v := learnID.String
		g.SourceLearningID = &v
	}
	g.Role = domainExp.GenerationRole(role)
	g.EngineType = domainExp.EngineType(eng)
	g.IsFrozen = frozenInt == 1
	g.Status = domainExp.GenerationStatus(st)
	g.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	g.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
	return &g, nil
}

func (r *ExperimentRepository) SaveComparison(ctx context.Context, cmp *domainExp.ExperimentComparison) error {
	now := clock.Now(ctx)
	if cmp.CreatedAt.IsZero() {
		cmp.CreatedAt = now
	}
	_, err := r.db.RawDB().ExecContext(ctx, `
		INSERT INTO experiment_comparisons (
			id, experiment_id, baseline_generation_id, candidate_generation_id, as_of,
			sample_count, performance_comparison_json, autonomy_comparison_json,
			regime_verdict, summary, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, cmp.ID, cmp.ExperimentID, cmp.BaselineGenerationID, cmp.CandidateGenerationID,
		cmp.AsOf.Format(time.RFC3339), cmp.SampleCount, cmp.PerformanceComparisonJSON,
		cmp.AutonomyComparisonJSON, string(cmp.RegimeVerdict), cmp.Summary,
		cmp.CreatedAt.Format(time.RFC3339))
	return err
}

func (r *ExperimentRepository) ListComparisons(ctx context.Context, experimentID string, limit int) ([]domainExp.ExperimentComparison, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.RawDB().QueryContext(ctx, `
		SELECT id, experiment_id, baseline_generation_id, candidate_generation_id, as_of,
			sample_count, performance_comparison_json, autonomy_comparison_json,
			regime_verdict, summary, created_at
		FROM experiment_comparisons WHERE experiment_id = ?
		ORDER BY created_at DESC LIMIT ?
	`, experimentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domainExp.ExperimentComparison
	for rows.Next() {
		var c domainExp.ExperimentComparison
		var ao, rv, ca string
		if err := rows.Scan(
			&c.ID, &c.ExperimentID, &c.BaselineGenerationID, &c.CandidateGenerationID, &ao,
			&c.SampleCount, &c.PerformanceComparisonJSON, &c.AutonomyComparisonJSON,
			&rv, &c.Summary, &ca,
		); err != nil {
			return nil, err
		}
		c.AsOf, _ = time.Parse(time.RFC3339, ao)
		c.RegimeVerdict = domainExp.RegimeVerdict(rv)
		c.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		out = append(out, c)
	}
	return out, rows.Err()
}
