package models

import (
	"history-api/internal/dtos/response"
	"time"
)

type WikiEntity struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	IsDeleted bool       `json:"is_deleted"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

func (w *WikiEntity) ToResponse() *response.WikiResponse {
	if w == nil {
		return nil
	}
	return &response.WikiResponse{
		ID:        w.ID,
		Title:     w.Title,
		Content:   w.Content,
		IsDeleted: w.IsDeleted,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
}

func WikisEntityToResponse(ws []*WikiEntity) []*response.WikiResponse {
	out := make([]*response.WikiResponse, 0)
	if ws == nil {
		return out
	}
	for _, w := range ws {
		if w == nil {
			continue
		}
		out = append(out, w.ToResponse())
	}
	return out
}
