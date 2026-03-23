package response

import (
	"history-api/pkg/constant"

	"github.com/golang-jwt/jwt/v5"
)

type CommonResponse struct {
	Status  bool   `json:"status"`
	Data    any    `json:"data"`
	Message string `json:"message"`
}

type JWTClaims struct {
	UId   string          `json:"uid"`
	Roles []constant.Role `json:"roles"`
	jwt.RegisteredClaims
}