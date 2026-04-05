package models

import "history-api/pkg/constants"

type TokenEntity struct {
	Email     string              `json:"email"`
	Token     string              `json:"token"`
	TokenType constants.TokenType `json:"token_type"`
}

type TokenUploadEntity struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	StorageKey   string `json:"storage_key"`
	OriginalName string `json:"original_name"`
	MimeType     string `json:"mime_type"`
	Size         int64  `json:"size"`
	FileMetadata []byte `json:"file_metadata"`
}

type OAuthState struct {
	State       string `json:"state"`
	RedirectURL string `json:"redirect"`
}
