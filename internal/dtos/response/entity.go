package response

import "time"

type EntityResponse struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Slug         string     `json:"slug,omitempty"`
	Description  string     `json:"description,omitempty"`
	ProjectID    string     `json:"project_id"`
	Status       *int16     `json:"status,omitempty"`
	TimeStart    *int32     `json:"time_start,omitempty"`
	TimeEnd      *int32     `json:"time_end,omitempty"`
	IsDeleted    bool       `json:"is_deleted,omitempty"`
	CreatedAt    *time.Time `json:"created_at,omitempty"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}
