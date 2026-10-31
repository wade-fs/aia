package unit

import (
	"testing"

	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/task"
)

func TestTaskStateTransitions(t *testing.T) {
	// Valid transitions
	validCases := []struct {
		from task.TaskStatus
		to   task.TaskStatus
	}{
		{task.StatusCreated, task.StatusReady},
		{task.StatusCreated, task.StatusWaitingDependency},
		{task.StatusCreated, task.StatusWaitingApproval},
		{task.StatusReady, task.StatusRunning},
		{task.StatusRunning, task.StatusCompleted},
		{task.StatusRunning, task.StatusFailed},
		{task.StatusFailed, task.StatusReady}, // retry
		{task.StatusRunning, task.StatusWaitingApproval},
		{task.StatusWaitingApproval, task.StatusReady},
	}

	for _, tc := range validCases {
		if err := task.ValidateTransition(tc.from, tc.to); err != nil {
			t.Errorf("expected valid transition from %s to %s, got: %v", tc.from, tc.to, err)
		}
	}

	// Invalid transitions
	invalidCases := []struct {
		from task.TaskStatus
		to   task.TaskStatus
	}{
		{task.StatusCompleted, task.StatusRunning},
		{task.StatusCompleted, task.StatusReady},
		{task.StatusCancelled, task.StatusRunning},
		{task.StatusCreated, task.StatusCompleted},
	}

	for _, tc := range invalidCases {
		if err := task.ValidateTransition(tc.from, tc.to); err == nil {
			t.Errorf("expected invalid transition from %s to %s, got nil", tc.from, tc.to)
		}
	}
}

func TestDependencyCycleDetection(t *testing.T) {
	// Graph: A -> B -> C
	graph := map[domain.ID][]domain.ID{
		"A": {"B"},
		"B": {"C"},
	}

	// Adding C -> A should create a cycle
	if !task.DetectCycle(graph, "C", "A") {
		t.Errorf("expected cycle detected for C -> A")
	}

	// Adding D -> C should not create a cycle
	if task.DetectCycle(graph, "D", "C") {
		t.Errorf("expected no cycle for D -> C")
	}
}
