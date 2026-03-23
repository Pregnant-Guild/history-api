package models

import (
	"history-api/pkg/convert"
	"history-api/pkg/dtos/response"

	"github.com/jackc/pgx/v5/pgtype"
)

type TokenEntity struct {
	ID        pgtype.UUID        `json:"id"`
	UserID    pgtype.UUID        `json:"user_id"`
	Token     string             `json:"token"`
	TokenType int16              `json:"token_type"`
	ExpiresAt pgtype.Timestamptz `json:"expires_at"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
}

func (t *TokenEntity) ToResponse() *response.TokenResponse {
	return &response.TokenResponse{
		ID:        convert.UUIDToString(t.ID),
		UserID:    convert.UUIDToString(t.UserID),
		TokenType: t.TokenType,
		ExpiresAt: convert.TimeToPtr(t.ExpiresAt),
		CreatedAt: convert.TimeToPtr(t.CreatedAt),
	}
}

func TokensEntityToResponse(ts []*TokenEntity) []*response.TokenResponse {
	out := make([]*response.TokenResponse, len(ts))
	for i := range ts {
		out[i] = ts[i].ToResponse()
	}
	return out
}
