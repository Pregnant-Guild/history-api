package models

import (
	"history-api/internal/dtos/response"
	"time"
)

type EntityEntity struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Slug         string     `json:"slug"`
	Description  string     `json:"description"`
	ProjectID    string     `json:"project_id"`
	Status       *int16     `json:"status"`
	IsDeleted    bool       `json:"is_deleted"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

func (e *EntityEntity) ToResponse() *response.EntityResponse {
	if e == nil {
		return nil
	}
	return &response.EntityResponse{
		ID:           e.ID,
		Name:         e.Name,
		Slug:         e.Slug,
		Description:  e.Description,
		ProjectID:    e.ProjectID,
		Status:       e.Status,
		IsDeleted:    e.IsDeleted,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}

func EntitiesEntityToResponse(es []*EntityEntity) []*response.EntityResponse {
	out := make([]*response.EntityResponse, 0)
	if es == nil {
		return out
	}
	for _, e := range es {
		if e == nil {
			continue
		}
		out = append(out, e.ToResponse())
	}
	return out
}
