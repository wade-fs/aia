package retrieval

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/wade/aia/internal/config"
	"github.com/wade/aia/internal/domain"
	"github.com/wade/aia/internal/domain/memory"
	"github.com/wade/aia/internal/storage/sqlite/repository"
)

type ScoredMemory struct {
	Memory memory.Memory
	Score  float64
}

type Retriever struct {
	repo *repository.MemoryRepository
	cfg  config.MemoryConfig
}

func NewRetriever(repo *repository.MemoryRepository, cfg config.MemoryConfig) *Retriever {
	return &Retriever{repo: repo, cfg: cfg}
}

func (r *Retriever) Retrieve(ctx context.Context, query string, projectID, agentID domain.ID) ([]memory.Memory, error) {
	memories, err := r.repo.Search(ctx, repository.MemoryQuery{
		ProjectID: projectID,
		AgentID:   agentID,
		Keyword:   query,
		Limit:     50,
	})
	if err != nil {
		return nil, err
	}

	keywords := strings.Fields(strings.ToLower(query))
	now := time.Now().UTC()

	var scored []ScoredMemory
	for _, m := range memories {
		score := m.Confidence
		titleLower := strings.ToLower(m.Title)
		contentLower := strings.ToLower(m.Content)

		matchCount := 0
		for _, kw := range keywords {
			if strings.Contains(titleLower, kw) {
				matchCount += 3
			}
			if strings.Contains(contentLower, kw) {
				matchCount += 1
			}
		}

		score += float64(matchCount) * 0.5

		// Recency boost (within 7 days gets +0.2)
		if now.Sub(m.CreatedAt) < 7*24*time.Hour {
			score += 0.2
		}

		scored = append(scored, ScoredMemory{Memory: m, Score: score})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	maxItems := r.cfg.MaxContextItems
	if maxItems <= 0 {
		maxItems = 10
	}
	maxBytes := r.cfg.MaxContextBytes
	if maxBytes <= 0 {
		maxBytes = 8192
	}

	var result []memory.Memory
	currentBytes := 0

	for _, sm := range scored {
		if len(result) >= maxItems {
			break
		}
		itemBytes := len(sm.Memory.Title) + len(sm.Memory.Content)
		if currentBytes+itemBytes > maxBytes && len(result) > 0 {
			break
		}
		result = append(result, sm.Memory)
		currentBytes += itemBytes
		_ = r.repo.TouchLastUsed(ctx, sm.Memory.ID)
	}

	return result, nil
}

func FormatContext(memories []memory.Memory) string {
	if len(memories) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("### Relevant Long-Term Memory Context:\n")
	for i, m := range memories {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s: %s\n", i+1, m.Type, m.Title, m.Content))
	}
	sb.WriteString("\n")
	return sb.String()
}
