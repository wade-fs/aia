package problem

import "context"

// Repository defines the persistence interface for Problem, ProblemModel, and AnalysisRecord entities (AIA-22 §6, §12).
type Repository interface {
	Create(ctx context.Context, p *Problem) error
	Get(ctx context.Context, id string) (*Problem, error)
	Update(ctx context.Context, p *Problem) error
	ListByProject(ctx context.Context, projectID string) ([]*Problem, error)
	ListChildren(ctx context.Context, parentID string) ([]*Problem, error)

	SaveModel(ctx context.Context, model *ProblemModel) error
	GetModel(ctx context.Context, problemID string, version int) (*ProblemModel, error)
	GetLatestModel(ctx context.Context, problemID string) (*ProblemModel, error)
	ListModels(ctx context.Context, problemID string) ([]*ProblemModel, error)

	CreateAnalysisRecord(ctx context.Context, rec *AnalysisRecord) error
	ListAnalysisRecords(ctx context.Context, problemID string) ([]*AnalysisRecord, error)
	GetAnalysisRecord(ctx context.Context, id string) (*AnalysisRecord, error)
}
