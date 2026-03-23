package middlewares

import (
	"history-api/pkg/config"
	"history-api/pkg/constant"
	"history-api/pkg/dtos/response"
	"slices"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
)

func JwtAccess() fiber.Handler {
	jwtSecret, err := config.GetConfig("JWT_SECRET")
	if err != nil {
		return nil
	}

	return jwtware.New(jwtware.Config{
		SigningKey:     jwtware.SigningKey{Key: []byte(jwtSecret)},
		ErrorHandler:   jwtError,
		SuccessHandler: jwtSuccess,
		Extractor:      extractors.FromAuthHeader("Bearer"),
		Claims:         &response.JWTClaims{},
	})
}

func JwtRefresh() fiber.Handler {
	jwtRefreshSecret, err := config.GetConfig("JWT_REFRESH_SECRET")
	if err != nil {
		return nil
	}

	return jwtware.New(jwtware.Config{
		SigningKey:     jwtware.SigningKey{Key: []byte(jwtRefreshSecret)},
		ErrorHandler:   jwtError,
		SuccessHandler: jwtSuccess,
		Extractor:      extractors.FromAuthHeader("Bearer"),
		Claims:         &response.JWTClaims{},
	})
}

func jwtSuccess(c fiber.Ctx) error {
	user := jwtware.FromContext(c)
	unauthorized := func() error {
		return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
			Status:  false,
			Message: "Invalid or missing token",
		})
	}

	if user == nil {
		return unauthorized()
	}

	claims, ok := user.Claims.(*response.JWTClaims)
	if !ok {
		return unauthorized()
	}

	if slices.Contains(claims.Roles, constant.BANNED) {
		return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{
			Status:  false,
			Message: "User account is banned",
		})
	}

	c.Locals("uid", claims.UId)
	c.Locals("user_claims", claims)

	return c.Next()
}
func jwtError(c fiber.Ctx, err error) error {
	if err.Error() == "Missing or malformed JWT" {
		return c.Status(fiber.StatusBadRequest).
			JSON(response.CommonResponse{
				Status:  false,
				Message: "Missing or malformed JWT",
			})
	}
	return c.Status(fiber.StatusUnauthorized).
		JSON(response.CommonResponse{
			Status:  false,
			Message: "Invalid or expired JWT",
		})
}
