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

type RagIndexTask struct {
	ProjectID       string           `json:"project_id"`
	DeleteWikiIDs   []string         `json:"delete_wiki_ids"`
	DeleteEntityIDs []string         `json:"delete_entity_ids"`
	Wikis           []*RagWikiItem   `json:"wikis"`
	Entities        []*RagEntityItem `json:"entities"`
}

type RagWikiItem struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Doc    string `json:"doc"`
	Source string `json:"source"`
}

type RagEntityItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
}
