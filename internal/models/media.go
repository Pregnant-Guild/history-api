package models

import (
	"history-api/internal/dtos/response"
	"time"
)

type MediaEntity struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	StorageKey   string     `json:"storage_key"`
	OriginalName string     `json:"original_name"`
	MimeType     string     `json:"mime_type"`
	Size         int64      `json:"size"`
	FileMetadata []byte     `json:"file_metadata"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

type MediaStorageEntity struct {
	ID         string `json:"id"`
	StorageKey string `json:"storage_key"`
}

func (e * MediaEntity) ToStorageEntity() *MediaStorageEntity {
	return &MediaStorageEntity{
		ID:         e.ID,
		StorageKey: e.StorageKey,
	}
}

func (e *MediaEntity) ToResponse() *response.MediaResponse {
	return &response.MediaResponse{
		ID:           e.ID,
		UserID:       e.UserID,
		StorageKey:   e.StorageKey,
		OriginalName: e.OriginalName,
		MimeType:     e.MimeType,
		Size:         e.Size,
		FileMetadata: e.FileMetadata,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}

func MediaEntitiesToResponse(entities []*MediaEntity) []*response.MediaResponse {
	responses := make([]*response.MediaResponse, len(entities))
	for i, entity := range entities {
		responses[i] = entity.ToResponse()
	}
	return responses
}
