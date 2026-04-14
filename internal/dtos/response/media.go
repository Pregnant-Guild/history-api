package response

import (
	"encoding/json"
	"time"
)

type PreSignedResponse struct {
	TokenID       string            `json:"token_id"`
	UploadUrl     string            `json:"upload_url"`
	StorageKey    string            `json:"storage_key"`
	SignedHeaders map[string]string `json:"signed_headers"`
}

type MediaResponse struct {
	ID           string          `json:"id"`
	UserID       string          `json:"user_id"`
	StorageKey   string          `json:"storage_key"`
	OriginalName string          `json:"original_name"`
	MimeType     string          `json:"mime_type"`
	Size         int64           `json:"size"`
	FileMetadata json.RawMessage `json:"file_metadata"`
	CreatedAt    *time.Time      `json:"created_at"`
	UpdatedAt    *time.Time      `json:"updated_at"`
}

type MediaSimpleResponse struct {
	ID           string          `json:"id"`
	StorageKey   string          `json:"storage_key"`
	OriginalName string          `json:"original_name"`
	MimeType     string          `json:"mime_type"`
	Size         int64           `json:"size"`
	FileMetadata json.RawMessage `json:"file_metadata"`
	CreatedAt    *time.Time      `json:"created_at"`
}
