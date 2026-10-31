package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/agent-runtime/internal/storage"
)

type Logger struct {
	storage *storage.DB
}

func NewLogger(db *storage.DB) *Logger {
	return &Logger{storage: db}
}

func (l *Logger) Log(level, action, entityType, entityID, detail string) error {
	now := time.Now()
	// Format to stdout
	timestampStr := now.Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] [AUDIT] [%s] %s | entity=%s/%s | %s\n",
		timestampStr, level, action, entityType, entityID, detail)

	if l.storage != nil {
		record := &storage.AuditLogRecord{
			ID:         uuid.New().String(),
			Level:      level,
			Action:     action,
			EntityType: entityType,
			EntityID:   entityID,
			Detail:     detail,
			CreatedAt:  now,
		}
		if err := l.storage.SaveAuditLog(context.Background(), record); err != nil {
			return fmt.Errorf("failed to persist audit log: %w", err)
		}
	}
	return nil
}

func (l *Logger) LogInfo(action, entityType, entityID, detail string) error {
	return l.Log("INFO", action, entityType, entityID, detail)
}

func (l *Logger) LogWarn(action, entityType, entityID, detail string) error {
	return l.Log("WARN", action, entityType, entityID, detail)
}

func (l *Logger) LogError(action, entityType, entityID, detail string) error {
	return l.Log("ERROR", action, entityType, entityID, detail)
}
