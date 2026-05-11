package response

import (
	"time"
)

type WikiContentSample struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	CreatedAt *time.Time `json:"created_at"`
}

type WikiResponse struct {
	ID            string              `json:"id"`
	Title         string              `json:"title,omitempty"`
	Slug          string              `json:"slug,omitempty"`
	ContentSample []WikiContentSample `json:"content_sample,omitempty"`
	ProjectID     string              `json:"project_id"`
	IsDeleted     bool                `json:"is_deleted,omitempty"`
	CreatedAt     *time.Time          `json:"created_at,omitempty"`
	UpdatedAt     *time.Time          `json:"updated_at,omitempty"`
}

type WikiContentResponse struct {
	ID        string     `json:"id"`
	WikiID    string     `json:"wiki_id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	CreatedAt *time.Time `json:"created_at"`
}
