package models

import (
	"history-api/internal/dtos/response"
	"time"
)

type StatisticEntity struct {
	ID                string     `json:"id"`
	Date              time.Time  `json:"date"`
	TotalUsers        int32      `json:"total_users"`
	TotalProjects     int32      `json:"total_projects"`
	TotalCommits      int32      `json:"total_commits"`
	TotalSubmissions  int32      `json:"total_submissions"`
	TotalMedias       int32      `json:"total_medias"`
	TotalWikis        int32      `json:"total_wikis"`
	TotalEntities     int32      `json:"total_entities"`
	TotalGeometries   int32      `json:"total_geometries"`
	TotalStorageBytes int64      `json:"total_storage_bytes"`
	NewUsers          int32      `json:"new_users"`
	NewProjects       int32      `json:"new_projects"`
	NewCommits        int32      `json:"new_commits"`
	NewSubmissions    int32      `json:"new_submissions"`
	NewMedias         int32      `json:"new_medias"`
	NewWikis          int32      `json:"new_wikis"`
	NewEntities       int32      `json:"new_entities"`
	NewGeometries     int32      `json:"new_geometries"`
	NewStorageBytes   int64      `json:"new_storage_bytes"`
	CreatedAt         *time.Time `json:"created_at"`
}

func (e *StatisticEntity) ToResponse() *response.StatisticResponse {
	if e == nil {
		return nil
	}

	dateStr := e.Date.Format("2006-01-02")
	var createdAt time.Time
	if e.CreatedAt != nil {
		createdAt = *e.CreatedAt
	}

	return &response.StatisticResponse{
		ID:                e.ID,
		Date:              dateStr,
		TotalUsers:        e.TotalUsers,
		TotalProjects:     e.TotalProjects,
		TotalCommits:      e.TotalCommits,
		TotalSubmissions:  e.TotalSubmissions,
		TotalMedias:       e.TotalMedias,
		TotalWikis:        e.TotalWikis,
		TotalEntities:     e.TotalEntities,
		TotalGeometries:   e.TotalGeometries,
		TotalStorageBytes: e.TotalStorageBytes,
		NewUsers:          e.NewUsers,
		NewProjects:       e.NewProjects,
		NewCommits:        e.NewCommits,
		NewSubmissions:    e.NewSubmissions,
		NewMedias:         e.NewMedias,
		NewWikis:          e.NewWikis,
		NewEntities:       e.NewEntities,
		NewGeometries:     e.NewGeometries,
		NewStorageBytes:   e.NewStorageBytes,
		CreatedAt:         createdAt,
	}
}
