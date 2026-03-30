package response

import "time"

type PreSignedResponse struct {
	UploadUrl     string            `json:"uploadUrl"`
	PublicUrl     string            `json:"publicUrl"`
	FileName      string            `json:"fileName"`
	MediaId       string            `json:"mediaId"`
	SignedHeaders map[string]string `json:"signedHeaders"`
}

type MediaResponse struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	StorageKey   string     `json:"storage_key"`
	OriginalName string     `json:"original_name"`
	MimeType     string     `json:"mime_type"`
	Size         int64      `json:"size"`
	TargetType   string     `json:"target_type"`
	TargetID     string     `json:"target_id"`
	FileMetadata []byte     `json:"file_metadata"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}
