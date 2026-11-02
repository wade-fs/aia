package plan

import "context"

// Repository defines the persistence interface for Plan and PlanChangeRecord entities (AIA-22 §6.4, §6.5).
type Repository interface {
	Create(ctx context.Context, p *Plan) error
	Get(ctx context.Context, id string) (*Plan, error)
	GetByProblemAndVersion(ctx context.Context, problemID string, version int) (*Plan, error)
	GetLatestByProblem(ctx context.Context, problemID string) (*Plan, error)
	ListByProblem(ctx context.Context, problemID string) ([]*Plan, error)
	UpdateStatus(ctx context.Context, id string, status PlanStatus) error

	RecordChange(ctx context.Context, change *PlanChangeRecord) error
	ListChanges(ctx context.Context, planID string) ([]*PlanChangeRecord, error)
}
