package models

import (
	"history-api/internal/dtos/response"
	"time"
)

type WikiContentSample struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	CreatedAt *time.Time `json:"created_at"`
}

type WikiEntity struct {
	ID            string              `json:"id"`
	Title         string              `json:"title"`
	Slug          string              `json:"slug"`
	ContentSample []WikiContentSample `json:"content_sample"`
	ProjectID     string              `json:"project_id"`
	IsDeleted     bool                `json:"is_deleted"`
	CreatedAt     *time.Time          `json:"created_at"`
	UpdatedAt     *time.Time          `json:"updated_at"`
}

func (w *WikiEntity) ToResponse() *response.WikiResponse {
	if w == nil {
		return nil
	}

	contentSample := make([]response.WikiContentSample, 0, len(w.ContentSample))
	for _, c := range w.ContentSample {
		contentSample = append(contentSample, response.WikiContentSample{
			ID:        c.ID,
			Title:     c.Title,
			CreatedAt: c.CreatedAt,
		})
	}

	return &response.WikiResponse{
		ID:            w.ID,
		Title:         w.Title,
		Slug:          w.Slug,
		ContentSample: contentSample,
		ProjectID:     w.ProjectID,
		IsDeleted:     w.IsDeleted,
		CreatedAt:     w.CreatedAt,
		UpdatedAt:     w.UpdatedAt,
	}
}

func WikisEntityToResponse(ws []*WikiEntity) []*response.WikiResponse {
	if ws == nil {
		return []*response.WikiResponse{}
	}
	out := make([]*response.WikiResponse, 0, len(ws))
	for _, w := range ws {
		if w == nil {
			continue
		}
		out = append(out, w.ToResponse())
	}
	return out
}

type WikiContentEntity struct {
	ID        string     `json:"id"`
	WikiID    string     `json:"wiki_id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	Preview   string     `json:"preview"`
	IsDeleted bool       `json:"is_deleted"`
	CreatedAt *time.Time `json:"created_at"`
}
