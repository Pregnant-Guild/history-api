package response

type TokenResponse struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	TokenType int16   `json:"token_type"`
	ExpiresAt *string `json:"expires_at"`
	CreatedAt *string `json:"created_at"`
}