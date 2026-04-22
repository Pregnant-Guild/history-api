package response

import "time"

type WikiResponse struct {
	ID        string     `json:"id"`
	Title     string     `json:"title,omitempty"`
	Content   string     `json:"content,omitempty"`
	IsDeleted bool       `json:"is_deleted,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}
