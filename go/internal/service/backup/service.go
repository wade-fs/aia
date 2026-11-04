package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/internal/eventbus"
	"github.com/wade/aia/internal/storage/sqlite"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

// Service manages online backup, integrity verification, and recovery procedures (AIA-19 §12, §13).
type Service struct {
	backupDir string
	db        *sqlite.DB
	dbPath    string
	cfgPath   string
	auditRepo *repository.AuditRepository
	bus       eventbus.Publisher
}

func NewService(
	backupDir string,
	db *sqlite.DB,
	dbPath string,
	cfgPath string,
	auditRepo *repository.AuditRepository,
	bus eventbus.Publisher,
) *Service {
	return &Service{
		backupDir: backupDir,
		db:        db,
		dbPath:    dbPath,
		cfgPath:   cfgPath,
		auditRepo: auditRepo,
		bus:       bus,
	}
}

// CreateBackup generates an online, point-in-time consistent backup snapshot (AIA-19 §12.2, §12.3).
func (s *Service) CreateBackup(ctx context.Context, note string) (*BackupManifest, error) {
	now := time.Now().UTC()
	backupID := fmt.Sprintf("backup-%s-%s", now.Format("20060102-150405"), uuid.New().String()[:6])
	destDir := filepath.Join(s.backupDir, backupID)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	destDB := filepath.Join(destDir, "aia.db")

	// 1. Point-in-time SQLite snapshot via VACUUM INTO or WAL checkpoint copy
	var schemaVer int
	if s.db != nil {
		_ = s.db.RawDB().QueryRowContext(ctx, "SELECT count(*) FROM schema_migrations").Scan(&schemaVer)

		// Checkpoint active WAL
		_, _ = s.db.RawDB().ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)")

		// Attempt VACUUM INTO for online snapshot
		_, vacErr := s.db.RawDB().ExecContext(ctx, fmt.Sprintf("VACUUM INTO '%s'", destDB))
		if vacErr != nil {
			// Fallback to direct file copy
			if s.dbPath != "" {
				if err := copyFile(s.dbPath, destDB); err != nil {
					return nil, fmt.Errorf("backup database copy failed: %w", err)
				}
			} else {
				return nil, fmt.Errorf("vacuum into failed: %w", vacErr)
			}
		}
	} else if s.dbPath != "" {
		if err := copyFile(s.dbPath, destDB); err != nil {
			return nil, fmt.Errorf("backup database copy failed: %w", err)
		}
	}

	// 2. Backup configuration file if exists
	destCfg := ""
	if s.cfgPath != "" {
		if _, err := os.Stat(s.cfgPath); err == nil {
			destCfg = filepath.Join(destDir, "config.yaml")
			_ = copyFile(s.cfgPath, destCfg)
		}
	}

	// 3. Compute Checksum SHA-256 and Database Size
	checksum, size, err := computeFileSHA256(destDB)
	if err != nil {
		return nil, fmt.Errorf("failed to compute backup checksum: %w", err)
	}

	included := []string{"sqlite_database", "event_history", "audit_logs", "continuity_state"}
	if destCfg != "" {
		included = append(included, "configuration")
	}

	manifest := &BackupManifest{
		BackupID:       backupID,
		CreatedAt:      now,
		AIAVersion:     "1.0.0",
		SchemaVersion:  schemaVer,
		IncludedData:   included,
		DatabaseFile:   "aia.db",
		ConfigFile:     filepath.Base(destCfg),
		ChecksumSHA256: checksum,
		DatabaseSize:   size,
		Note:           note,
		Status:         "COMPLETED",
	}

	// Write manifest.json
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(destDir, "manifest.json"), manifestBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to write backup manifest: %w", err)
	}

	// Audit logging
	if s.auditRepo != nil {
		det, _ := json.Marshal(map[string]any{
			"backup_id":       backupID,
			"checksum_sha256": checksum,
			"database_size":   size,
			"schema_version":  schemaVer,
			"note":            note,
		})
		risk := "LOW"
		_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:           uuid.New().String(),
			ActorType:    "SYSTEM",
			ActorID:      "backup-service",
			Action:       "BACKUP_CREATED",
			ResourceType: "BACKUP",
			ResourceID:   backupID,
			Outcome:      "SUCCESS",
			Details:      det,
			RiskTier:     &risk,
			CreatedAt:    now,
		})
	}

	if s.bus != nil {
		s.bus.Publish("backup.created", map[string]any{
			"backup_id": backupID,
			"timestamp": now,
		})
	}

	return manifest, nil
}

// ListBackups enumerates all available backup points in chronological order (AIA-19 §12.3).
func (s *Service) ListBackups(ctx context.Context) ([]*BackupManifest, error) {
	entries, err := os.ReadDir(s.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*BackupManifest{}, nil
		}
		return nil, err
	}

	var manifests []*BackupManifest
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		manifestPath := filepath.Join(s.backupDir, e.Name(), "manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		var m BackupManifest
		if err := json.Unmarshal(data, &m); err == nil {
			manifests = append(manifests, &m)
		}
	}

	sort.Slice(manifests, func(i, j int) bool {
		return manifests[i].CreatedAt.After(manifests[j].CreatedAt)
	})

	return manifests, nil
}

// GetBackup loads a specific backup manifest.
func (s *Service) GetBackup(ctx context.Context, backupID string) (*BackupManifest, error) {
	manifestPath := filepath.Join(s.backupDir, backupID, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrBackupNotFound
		}
		return nil, err
	}
	var m BackupManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("corrupted backup manifest: %w", err)
	}
	return &m, nil
}

// VerifyBackup checks checksum and database integrity of a backup point (AIA-19 §12.3, §12.4).
func (s *Service) VerifyBackup(ctx context.Context, backupID string) (*VerificationResult, error) {
	m, err := s.GetBackup(ctx, backupID)
	if err != nil {
		return nil, err
	}

	dbFile := filepath.Join(s.backupDir, backupID, m.DatabaseFile)
	res := &VerificationResult{
		BackupID:      backupID,
		Valid:         true,
		SchemaVersion: m.SchemaVersion,
		VerifiedAt:    time.Now().UTC(),
	}

	// 1. Checksum verification
	actualSHA, _, err := computeFileSHA256(dbFile)
	if err != nil {
		res.Valid = false
		res.Issues = append(res.Issues, fmt.Sprintf("cannot read backup db file: %v", err))
		return res, nil
	}
	if actualSHA != m.ChecksumSHA256 {
		res.Valid = false
		res.ChecksumMatches = false
		res.Issues = append(res.Issues, fmt.Sprintf("checksum mismatch: expected %s, got %s", m.ChecksumSHA256, actualSHA))
	} else {
		res.ChecksumMatches = true
	}

	// 2. SQLite integrity check
	checkDB, err := sql.Open("sqlite", dbFile)
	if err != nil {
		res.Valid = false
		res.SQLiteIntegrityOK = false
		res.Issues = append(res.Issues, fmt.Sprintf("failed to open database: %v", err))
		return res, nil
	}
	defer checkDB.Close()

	var integrityResult string
	if err := checkDB.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integrityResult); err != nil {
		res.Valid = false
		res.SQLiteIntegrityOK = false
		res.Issues = append(res.Issues, fmt.Sprintf("integrity_check query error: %v", err))
	} else if strings.ToLower(integrityResult) != "ok" {
		res.Valid = false
		res.SQLiteIntegrityOK = false
		res.Issues = append(res.Issues, fmt.Sprintf("integrity_check failed: %s", integrityResult))
	} else {
		res.SQLiteIntegrityOK = true
	}

	return res, nil
}

// RestoreBackup verifies and restores a backup snapshot into the target database path (AIA-19 §12.4).
func (s *Service) RestoreBackup(ctx context.Context, backupID string, targetDBPath string) error {
	if targetDBPath == "" {
		return ErrInvalidRestoreTarget
	}

	// Verify first
	ver, err := s.VerifyBackup(ctx, backupID)
	if err != nil {
		return err
	}
	if !ver.Valid {
		return fmt.Errorf("%w: %s", ErrIntegrityCheckFailed, strings.Join(ver.Issues, ", "))
	}

	srcDB := filepath.Join(s.backupDir, backupID, "aia.db")

	// Safety: backup current target if exists
	if _, err := os.Stat(targetDBPath); err == nil {
		safetyCopy := targetDBPath + fmt.Sprintf(".pre-restore-%d", time.Now().Unix())
		_ = copyFile(targetDBPath, safetyCopy)
	}

	// Restore database
	if err := copyFile(srcDB, targetDBPath); err != nil {
		return fmt.Errorf("failed to restore database file: %w", err)
	}

	// Restore config if available
	srcCfg := filepath.Join(s.backupDir, backupID, "config.yaml")
	if s.cfgPath != "" {
		if _, err := os.Stat(srcCfg); err == nil {
			_ = copyFile(srcCfg, s.cfgPath)
		}
	}

	now := time.Now().UTC()
	if s.auditRepo != nil {
		det, _ := json.Marshal(map[string]any{
			"backup_id":      backupID,
			"target_db_path": targetDBPath,
			"schema_version": ver.SchemaVersion,
		})
		risk := "HIGH"
		_ = s.auditRepo.Append(ctx, &repository.AuditLogEntry{
			ID:           uuid.New().String(),
			ActorType:    "SYSTEM",
			ActorID:      "backup-service",
			Action:       "BACKUP_RESTORED",
			ResourceType: "DATABASE",
			ResourceID:   targetDBPath,
			Outcome:      "SUCCESS",
			Details:      det,
			RiskTier:     &risk,
			CreatedAt:    now,
		})
	}

	if s.bus != nil {
		s.bus.Publish("backup.restored", map[string]any{
			"backup_id": backupID,
			"target_db": targetDBPath,
			"timestamp": now,
		})
	}

	return nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func computeFileSHA256(filePath string) (string, int64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return "", 0, err
	}

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", 0, err
	}

	return hex.EncodeToString(h.Sum(nil)), fi.Size(), nil
}
