package backup

import (
	"errors"
	"time"
)

var (
	ErrBackupNotFound       = errors.New("backup not found")
	ErrIntegrityCheckFailed = errors.New("backup integrity verification failed: checksum mismatch or corrupted database")
	ErrInvalidRestoreTarget = errors.New("target database path cannot be empty")
)

// BackupManifest records backup metadata and integrity provenance (AIA-19 §12.3).
type BackupManifest struct {
	BackupID       string    `json:"backup_id"`
	CreatedAt      time.Time `json:"created_at"`
	AIAVersion     string    `json:"aia_version"`
	SchemaVersion  int       `json:"schema_version"`
	IncludedData   []string  `json:"included_data"`
	DatabaseFile   string    `json:"database_file"`
	ConfigFile     string    `json:"config_file,omitempty"`
	ChecksumSHA256 string    `json:"checksum_sha256"`
	DatabaseSize   int64     `json:"database_size"`
	Note           string    `json:"note,omitempty"`
	Status         string    `json:"status"` // COMPLETED, FAILED
}

// VerificationResult details outcome of a backup verification inspection.
type VerificationResult struct {
	BackupID          string    `json:"backup_id"`
	Valid             bool      `json:"valid"`
	ChecksumMatches   bool      `json:"checksum_matches"`
	SQLiteIntegrityOK bool      `json:"sqlite_integrity_ok"`
	SchemaVersion     int       `json:"schema_version"`
	Issues            []string  `json:"issues,omitempty"`
	VerifiedAt        time.Time `json:"verified_at"`
}
