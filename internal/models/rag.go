package models

import (
	"time"
)

type RagChunk struct {
	ID         string    `json:"id"`
	SourceType string    `json:"source_type"`
	SourceID   string    `json:"source_id"`
	ProjectID  string    `json:"project_id"`
	ChunkIndex int32     `json:"chunk_index"`
	Content    string    `json:"content"`
	Similarity float64   `json:"similarity,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
