package response

import (
	"encoding/json"
	"time"
)

type WikiResponse struct {
	ID        string          `json:"id"`
	Title     string          `json:"title,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	ProjectID string          `json:"project_id"`
	IsDeleted bool            `json:"is_deleted,omitempty"`
	CreatedAt *time.Time      `json:"created_at,omitempty"`
	UpdatedAt *time.Time      `json:"updated_at,omitempty"`
}
