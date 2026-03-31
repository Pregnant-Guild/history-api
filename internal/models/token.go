package models

import "history-api/pkg/constants"

type TokenEntity struct {
	Email     string              `json:"email"`
	Token     string              `json:"token"`
	TokenType constants.TokenType `json:"token_type"`
}

type OAuthState struct {
	State       string `json:"state"`
	RedirectURL string `json:"redirect"`
}
