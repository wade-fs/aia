package artifact

import (
	"time"

	"github.com/wade/aia/internal/domain"
)

type Artifact struct {
	ID          domain.ID `json:"id"`
	TaskID      domain.ID `json:"task_id"`
	ExecutionID domain.ID `json:"execution_id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	MediaType   string    `json:"media_type"`
	SHA256      string    `json:"sha256"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
}
