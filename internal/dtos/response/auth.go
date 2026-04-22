package response

type AuthResponse struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type VerifyTokenResponse struct {
	TokenID string `json:"token_id"`
}
