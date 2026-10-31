package aider

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/wade/aia/agent-runtime/internal/worker"
)

type Adapter struct {
	binaryPath   string
	defaultModel string
	timeout      time.Duration

	mu        sync.RWMutex
	processes map[string]*exec.Cmd
	exitCodes map[string]int
}

func NewAdapter(binaryPath, defaultModel string, timeout time.Duration) *Adapter {
	if binaryPath == "" {
		binaryPath = "aider"
	}
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	return &Adapter{
		binaryPath:   binaryPath,
		defaultModel: defaultModel,
		timeout:      timeout,
		processes:    make(map[string]*exec.Cmd),
		exitCodes:    make(map[string]int),
	}
}

func (a *Adapter) Initialize(ctx context.Context) error {
	return a.HealthCheck(ctx)
}

func (a *Adapter) HealthCheck(ctx context.Context) error {
	if strings.Contains(a.binaryPath, string(filepath.Separator)) {
		info, err := os.Stat(a.binaryPath)
		if err != nil {
			return fmt.Errorf("aider binary not found at %s: %w", a.binaryPath, err)
		}
		if info.IsDir() {
			return fmt.Errorf("aider binary path %s is a directory", a.binaryPath)
		}
		return nil
	}

	_, err := exec.LookPath(a.binaryPath)
	if err != nil {
		return fmt.Errorf("aider binary %q not found in PATH: %w", a.binaryPath, err)
	}
	return nil
}

func (a *Adapter) Execute(ctx context.Context, req worker.WorkerRequest) (*worker.WorkerResult, error) {
	executionID := req.ExecutionID
	if executionID == "" {
		executionID = uuid.New().String()
	}

	// 1. Create temporary message file for task description
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("aider-msg-%s-*.txt", executionID))
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary message file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(req.Description); err != nil {
		_ = tmpFile.Close()
		return nil, fmt.Errorf("failed to write description to message file: %w", err)
	}
	_ = tmpFile.Close()

	// 2. Setup context timeout
	timeout := a.timeout
	if req.Timeout > 0 {
		timeout = req.Timeout
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 3. Prepare aider command arguments
	args := []string{
		"--message-file", tmpFile.Name(),
		"--yes",
		"--no-auto-commits",
	}

	model := req.Model
	if model == "" {
		model = a.defaultModel
	}
	if model != "" {
		args = append(args, "--model", model)
	}

	cmd := exec.CommandContext(execCtx, a.binaryPath, args...)
	cmd.Dir = req.WorkspacePath

	// Environment configuration
	cmd.Env = os.Environ()
	for k, v := range req.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	startTime := time.Now()

	// 4. Start the command
	if err := cmd.Start(); err != nil {
		endTime := time.Now()
		return &worker.WorkerResult{
			ExecutionID: executionID,
			ExitCode:    -1,
			Stdout:      stdoutBuf.String(),
			Stderr:      stderrBuf.String(),
			StartTime:   startTime,
			EndTime:     endTime,
			Summary:     fmt.Sprintf("Failed to start aider: %v", err),
		}, fmt.Errorf("failed to start aider command: %w", err)
	}

	a.mu.Lock()
	a.processes[executionID] = cmd
	a.mu.Unlock()

	// Notify caller with PID
	if req.OnStart != nil && cmd.Process != nil {
		req.OnStart(cmd.Process.Pid)
	}

	// 5. Wait for command completion
	waitErr := cmd.Wait()
	endTime := time.Now()

	exitCode := 0
	if waitErr != nil {
		if exitError, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
	}

	a.mu.Lock()
	a.exitCodes[executionID] = exitCode
	delete(a.processes, executionID)
	a.mu.Unlock()

	summary := fmt.Sprintf("Aider finished with exit code %d in %v", exitCode, endTime.Sub(startTime))
	if exitCode != 0 {
		summary = fmt.Sprintf("Aider failed with exit code %d: %v", exitCode, waitErr)
	}

	return &worker.WorkerResult{
		ExecutionID: executionID,
		ExitCode:    exitCode,
		Stdout:      stdoutBuf.String(),
		Stderr:      stderrBuf.String(),
		StartTime:   startTime,
		EndTime:     endTime,
		Summary:     summary,
	}, waitErr
}

func (a *Adapter) GetStatus(ctx context.Context, executionID string) (*worker.WorkerStatus, error) {
	a.mu.RLock()
	cmd, exists := a.processes[executionID]
	exitCode, hasExit := a.exitCodes[executionID]
	a.mu.RUnlock()

	if !exists {
		if hasExit {
			return &worker.WorkerStatus{
				ExecutionID: executionID,
				IsRunning:   false,
				ExitCode:    &exitCode,
			}, nil
		}
		return &worker.WorkerStatus{
			ExecutionID: executionID,
			IsRunning:   false,
		}, nil
	}

	if cmd.Process == nil {
		return &worker.WorkerStatus{
			ExecutionID: executionID,
			IsRunning:   false,
		}, nil
	}

	pid := cmd.Process.Pid
	// Check if process is still alive using Signal(0)
	err := cmd.Process.Signal(syscall.Signal(0))
	isRunning := err == nil

	return &worker.WorkerStatus{
		ExecutionID: executionID,
		IsRunning:   isRunning,
		PID:         pid,
	}, nil
}

func (a *Adapter) Cancel(ctx context.Context, executionID string) error {
	a.mu.RLock()
	cmd, exists := a.processes[executionID]
	a.mu.RUnlock()

	if !exists || cmd.Process == nil {
		return fmt.Errorf("no running process found for execution %s", executionID)
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM to process %d: %w", cmd.Process.Pid, err)
	}
	return nil
}
