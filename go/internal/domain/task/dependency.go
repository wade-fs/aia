package task

import (
	"github.com/wade/aia/internal/domain"
)

type Dependency struct {
	TaskID          domain.ID `json:"task_id"`
	DependsOnTaskID domain.ID `json:"depends_on_task_id"`
}

// DetectCycle checks if adding an edge (taskID -> dependsOn) introduces a cycle in the dependency graph.
// graph is represented as map[TaskID][]DependsOnTaskID
func DetectCycle(graph map[domain.ID][]domain.ID, from, to domain.ID) bool {
	visited := make(map[domain.ID]bool)
	recStack := make(map[domain.ID]bool)

	// Temporarily add edge
	tempGraph := make(map[domain.ID][]domain.ID)
	for k, v := range graph {
		tempGraph[k] = append([]domain.ID(nil), v...)
	}
	tempGraph[from] = append(tempGraph[from], to)

	var hasCycle func(node domain.ID) bool
	hasCycle = func(node domain.ID) bool {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range tempGraph[node] {
			if !visited[neighbor] {
				if hasCycle(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}

		recStack[node] = false
		return false
	}

	for node := range tempGraph {
		if !visited[node] {
			if hasCycle(node) {
				return true
			}
		}
	}
	return false
}
