package worker

import (
	"context"
	"time"
)

type Worker interface {
	Initialize(ctx context.Context) error
	Execute(ctx context.Context, req WorkerRequest) (*WorkerResult, error)
	GetStatus(ctx context.Context, executionID string) (*WorkerStatus, error)
	Cancel(ctx context.Context, executionID string) error
	HealthCheck(ctx context.Context) error
}

type WorkerRequest struct {
	TaskID        string
	ExecutionID   string
	WorkspacePath string
	Description   string
	Model         string
	Timeout       time.Duration
	Env           map[string]string
	OnStart       func(pid int)
}

type WorkerResult struct {
	ExecutionID string
	ExitCode    int
	Stdout      string
	Stderr      string
	StartTime   time.Time
	EndTime     time.Time
	Summary     string
}

type WorkerStatus struct {
	ExecutionID string
	IsRunning   bool
	PID         int
	ExitCode    *int
}
