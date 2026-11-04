package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/wade/aia/internal/service/backup"
	"github.com/wade/aia/internal/storage/sqlite/repository"
	"github.com/wade/aia/internal/telemetry/metrics"
	"github.com/wade/aia/internal/telemetry/tracing"
)

func TestBackupService_Lifecycle(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	auditRepo := repository.NewAuditRepository(db)
	backupDir := t.TempDir()
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("runtime:\n  workspace_dir: /tmp\n"), 0644); err != nil {
		t.Fatalf("failed to write dummy config: %v", err)
	}

	svc := backup.NewService(backupDir, db, "", cfgPath, auditRepo, nil)

	// 1. Create Backup
	manifest, err := svc.CreateBackup(ctx, "Automated pre-upgrade backup")
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}
	if manifest.Status != "COMPLETED" {
		t.Fatalf("expected status COMPLETED, got %s", manifest.Status)
	}
	if manifest.ChecksumSHA256 == "" {
		t.Fatalf("expected non-empty checksum")
	}

	// 2. List Backups
	backups, err := svc.ListBackups(ctx)
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("expected 1 backup, got %d", len(backups))
	}
	if backups[0].BackupID != manifest.BackupID {
		t.Errorf("expected backup ID %s, got %s", manifest.BackupID, backups[0].BackupID)
	}

	// 3. Verify Backup
	verRes, err := svc.VerifyBackup(ctx, manifest.BackupID)
	if err != nil {
		t.Fatalf("VerifyBackup failed: %v", err)
	}
	if !verRes.Valid || !verRes.ChecksumMatches || !verRes.SQLiteIntegrityOK {
		t.Fatalf("backup failed verification: %+v", verRes)
	}

	// 4. Restore Backup
	targetDB := filepath.Join(t.TempDir(), "restored.db")
	if err := svc.RestoreBackup(ctx, manifest.BackupID, targetDB); err != nil {
		t.Fatalf("RestoreBackup failed: %v", err)
	}
	if fi, err := os.Stat(targetDB); err != nil || fi.Size() == 0 {
		t.Fatalf("restored db file missing or empty: %v", err)
	}
}

func TestTracing_SpanLifecycle_And_ContextPropagation(t *testing.T) {
	tracer := tracing.NewTracer(100)
	ctx := context.Background()

	// 1. Start root span
	ctxRoot, rootSpan := tracer.Start(ctx, tracing.SpanAutonomyCycle)
	traceID := tracing.TraceIDFromContext(ctxRoot)
	if traceID == "" {
		t.Fatalf("expected non-empty trace ID in context")
	}
	if rootSpan.TraceID != traceID {
		t.Errorf("span trace ID %s != context trace ID %s", rootSpan.TraceID, traceID)
	}

	// 2. Start child span
	_, childSpan := tracer.Start(ctxRoot, tracing.SpanAutonomyDiscover)
	if childSpan.TraceID != traceID {
		t.Errorf("child span trace ID %s != root trace ID %s", childSpan.TraceID, traceID)
	}
	if childSpan.ParentSpanID != rootSpan.SpanID {
		t.Errorf("child parent ID %s != root span ID %s", childSpan.ParentSpanID, rootSpan.SpanID)
	}

	childSpan.SetAttribute("candidates_found", 3)
	childSpan.End()

	if childSpan.EndTime == nil || childSpan.DurationMs < 0 {
		t.Errorf("child span end time not recorded properly")
	}

	rootSpan.End()

	// 3. Retrieve by trace ID
	spans := tracer.GetSpansByTrace(traceID)
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans for trace, got %d", len(spans))
	}

	// 4. Test error recording
	_, errSpan := tracer.Start(ctx, "failed.operation")
	errSpan.RecordError(os.ErrNotExist)
	errSpan.End()
	if errSpan.Status != "ERROR" || errSpan.ErrorMessage != os.ErrNotExist.Error() {
		t.Errorf("expected ERROR status and error message on failed span")
	}
}

func TestMetrics_OperationalCollection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	taskRepo := repository.NewTaskRepository(db)
	execRepo := repository.NewExecutionRepository(db)
	reg := metrics.NewRegistry()

	reg.Incr("active_workers", 3)
	reg.SetGauge("test_gauge", 42.0)

	snap, err := reg.Collect(ctx, metrics.MetricCollectorDeps{
		TaskRepo: taskRepo,
		ExecRepo: execRepo,
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if snap == nil {
		t.Fatalf("expected non-nil snapshot")
	}
	if snap.Runtime.ActiveGoroutines <= 0 {
		t.Errorf("expected positive active goroutines, got %d", snap.Runtime.ActiveGoroutines)
	}
	if snap.Runtime.ActiveWorkers != 3 {
		t.Errorf("expected 3 active workers, got %d", snap.Runtime.ActiveWorkers)
	}
	if snap.Runtime.AllocMemoryBytes <= 0 {
		t.Errorf("expected positive alloc memory")
	}
}
