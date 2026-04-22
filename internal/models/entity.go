package models

import (
	"history-api/internal/dtos/response"
	"time"
)

type EntityEntity struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	ThumbnailUrl string     `json:"thumbnail_url"`
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
		Description:  e.Description,
		ThumbnailUrl: e.ThumbnailUrl,
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
