package response

import "time"

type TokenResponse struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	TokenType int16      `json:"token_type"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt *time.Time `json:"created_at"`
}
