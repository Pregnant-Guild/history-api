package response

import (
	"history-api/pkg/constants"

	"github.com/golang-jwt/jwt/v5"
)

type CommonResponse struct {
	Status  bool   `json:"status"`
	Data    any    `json:"data"`
	Message string `json:"message"`
}

type JWTClaims struct {
	UId          string           `json:"uid"`
	Roles        []constants.Role `json:"roles"`
	TokenVersion int32            `json:"token_version"`
	jwt.RegisteredClaims
}

type PaginatedResponse struct {
	Data       any  `json:"data"`
	Status     bool   `json:"status"`
	Message    string `json:"message"`
	Pagination struct {
		NextCursor string `json:"next_cursor"`
		HasMore    bool   `json:"has_more"`
	} `json:"pagination"`
}
