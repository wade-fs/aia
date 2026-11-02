package brief

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/domain"
	domainBrief "github.com/wade/aia/internal/domain/brief"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

// Service manages Project Briefs, field changes, readiness assessments, and workspace synchronization (AIA-21 §3.3).
type Service struct {
	briefRepo   *repository.GenericBriefRepository
	projectRepo *repository.ProjectRepository
	auditRepo   *repository.AuditRepository
}

// NewService creates a new BriefService.
func NewService(briefRepo *repository.GenericBriefRepository, projectRepo *repository.ProjectRepository) *Service {
	return &Service{
		briefRepo:   briefRepo,
		projectRepo: projectRepo,
	}
}

// SetAuditRepo attaches an audit repository for change tracking (AIA-21 §13.9).
func (s *Service) SetAuditRepo(auditRepo *repository.AuditRepository) {
	s.auditRepo = auditRepo
}

// InitializeBrief initializes a new Project Brief, applies domain catalog schema, and synchronizes to workspace brief.yaml.
func (s *Service) InitializeBrief(ctx context.Context, projectID string, domainName string, rawIntent string, workspaceRoot string) (*domainBrief.ProjectBrief, error) {
	catalog, err := domainBrief.GetCatalog(domainName)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain catalog: %w", err)
	}

	b := domainBrief.NewProjectBrief("brief-"+uuid.New().String()[:8], projectID, domainName, rawIntent)

	// Populate initial unknown fields from catalog
	for key, def := range catalog.Fields {
		var initialStatus domainBrief.FieldStatus = domainBrief.FieldStatusUnknown
		var initialVal any = nil
		var source string = "catalog_initialization"
		var rationale string = def.Description
		var confReq bool = false

		if def.DefaultValue != nil {
			initialStatus = domainBrief.FieldStatusDefaulted
			initialVal = def.DefaultValue
			source = "agent_default"
			rationale = fmt.Sprintf("預設值: %v (%s)", def.DefaultValue, def.Description)
			confReq = true
		}

		b.SetField(
			key,
			initialVal,
			initialStatus,
			source,
			0.5,
			rationale,
			confReq,
			"system",
			"",
			"",
		)
	}

	// Assess initial readiness
	res := domainBrief.AssessReadiness(b, catalog)
	b.Readiness = res.Status
	b.ReadinessRationale = res.Rationale
	b.BriefVersion = 1 // Canonical base version for newly initialized brief (AIA-21 §4.5)

	// Persist in database
	if s.briefRepo != nil {
		if err := s.briefRepo.Save(ctx, b); err != nil {
			return nil, fmt.Errorf("failed to save brief: %w", err)
		}
	}

	// Synchronize to workspace brief.yaml
	if workspaceRoot != "" {
		if err := s.SyncToWorkspace(b, workspaceRoot); err != nil {
			return nil, fmt.Errorf("failed to sync brief to workspace: %w", err)
		}
	}

	return b, nil
}

// UpdateField records a value update to a brief field, updates history, re-evaluates readiness, and syncs brief.yaml.
func (s *Service) UpdateField(
	ctx context.Context,
	projectID string,
	key string,
	val any,
	status domainBrief.FieldStatus,
	source string,
	confidence float64,
	rationale string,
	confirmationRequired bool,
	updatedBy string,
	relatedTaskID string,
	relatedClarificationID string,
	workspaceRoot string,
) (*domainBrief.ProjectBrief, error) {
	return s.UpdateFieldWithVersion(
		ctx, projectID, 0, key, val, status, source, confidence,
		rationale, confirmationRequired, updatedBy, relatedTaskID, relatedClarificationID, workspaceRoot,
	)
}

// UpdateFieldWithVersion updates a field checking optimistic lock expectedVersion (AIA-21 §12.3).
func (s *Service) UpdateFieldWithVersion(
	ctx context.Context,
	projectID string,
	expectedVersion int,
	key string,
	val any,
	status domainBrief.FieldStatus,
	source string,
	confidence float64,
	rationale string,
	confirmationRequired bool,
	updatedBy string,
	relatedTaskID string,
	relatedClarificationID string,
	workspaceRoot string,
) (*domainBrief.ProjectBrief, error) {
	b, err := s.briefRepo.GetByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch brief: %w", err)
	}
	if b == nil {
		return nil, fmt.Errorf("brief for project %s not found", projectID)
	}

	b.SetField(
		key,
		val,
		status,
		source,
		confidence,
		rationale,
		confirmationRequired,
		updatedBy,
		relatedTaskID,
		relatedClarificationID,
	)

	// Re-evaluate readiness
	if catalog, err := domainBrief.GetCatalog(b.Domain); err == nil {
		res := domainBrief.AssessReadiness(b, catalog)
		b.Readiness = res.Status
		b.ReadinessRationale = res.Rationale
	}

	// Save to database with optimistic version check
	if err := s.briefRepo.SaveWithVersionCheck(ctx, b, expectedVersion); err != nil {
		return nil, fmt.Errorf("failed to update brief in database: %w", err)
	}

	// Synchronize to workspace brief.yaml
	if workspaceRoot != "" {
		_ = s.SyncToWorkspace(b, workspaceRoot)
	}

	// Audit trail logging (AIA-21 §13.9)
	if s.auditRepo != nil {
		_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:           domain.ID("audit-" + uuid.New().String()[:8]),
			ActorType:    "HUMAN",
			ActorID:      updatedBy,
			Action:       "BRIEF_FIELD_UPDATED",
			ResourceType: "PROJECT_BRIEF",
			ResourceID:   projectID,
			Outcome:      "SUCCESS",
			CreatedAt:    time.Now().UTC(),
		})
	}

	return b, nil
}

// GetBrief retrieves a project brief by project ID.
func (s *Service) GetBrief(ctx context.Context, projectID string) (*domainBrief.ProjectBrief, error) {
	return s.briefRepo.GetByProjectID(ctx, projectID)
}

// ListFieldHistory retrieves chronological revisions for a specific field.
func (s *Service) ListFieldHistory(ctx context.Context, projectID string, fieldKey string) ([]domainBrief.FieldHistoryEntry, error) {
	if s.briefRepo == nil {
		return nil, fmt.Errorf("brief repository not configured")
	}
	return s.briefRepo.ListFieldHistory(ctx, projectID, fieldKey)
}

// SyncToWorkspace atomically serializes and writes brief.yaml to the project directory.
func (s *Service) SyncToWorkspace(b *domainBrief.ProjectBrief, workspaceRoot string) error {
	if err := os.MkdirAll(workspaceRoot, 0755); err != nil {
		return fmt.Errorf("failed to create workspace dir: %w", err)
	}

	yamlStr, err := b.ToYAML()
	if err != nil {
		return fmt.Errorf("failed to serialize brief: %w", err)
	}

	briefPath := filepath.Join(workspaceRoot, "brief.yaml")
	tmpPath := briefPath + ".tmp"

	if err := os.WriteFile(tmpPath, []byte(yamlStr), 0644); err != nil {
		return fmt.Errorf("failed to write tmp brief.yaml: %w", err)
	}

	if err := os.Rename(tmpPath, briefPath); err != nil {
		return fmt.Errorf("failed to atomically replace brief.yaml: %w", err)
	}

	return nil
}
