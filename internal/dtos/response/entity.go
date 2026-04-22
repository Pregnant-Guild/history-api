package response

import "time"

type EntityResponse struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Description  string     `json:"description,omitempty"`
	ThumbnailUrl string     `json:"thumbnail_url,omitempty"`
	IsDeleted    bool       `json:"is_deleted"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}
