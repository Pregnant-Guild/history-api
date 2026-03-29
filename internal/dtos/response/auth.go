package response

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type VerifyTokenResponse struct {
	TokenID string `json:"token_id"`
}
