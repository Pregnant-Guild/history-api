package middlewares

import (
	"history-api/internal/dtos/response"
	"history-api/internal/repositories"
	"history-api/pkg/config"
	"history-api/pkg/constants"
	"slices"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/jackc/pgx/v5/pgtype"
)

func JwtAccess(userRepo repositories.UserRepository) fiber.Handler {
	jwtSecret, err := config.GetConfig("JWT_SECRET")
	if err != nil {
		return nil
	}

	return jwtware.New(jwtware.Config{
		SigningKey:     jwtware.SigningKey{Key: []byte(jwtSecret)},
		ErrorHandler:   jwtError,
		SuccessHandler: jwtSuccess(userRepo),
		Extractor:      extractors.FromAuthHeader("Bearer"),
		Claims:         &response.JWTClaims{},
	})
}

func JwtRefresh(userRepo repositories.UserRepository) fiber.Handler {
	jwtRefreshSecret, err := config.GetConfig("JWT_REFRESH_SECRET")
	if err != nil {
		return nil
	}

	return jwtware.New(jwtware.Config{
		SigningKey:     jwtware.SigningKey{Key: []byte(jwtRefreshSecret)},
		ErrorHandler:   jwtError,
		SuccessHandler: jwtSuccess(userRepo),
		Extractor:      extractors.FromAuthHeader("Bearer"),
		Claims:         &response.JWTClaims{},
	})
}

func jwtSuccess(userRepo repositories.UserRepository) fiber.Handler {
	return func(c fiber.Ctx) error {
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

		if slices.Contains(claims.Roles, constants.BANNED) {
			return c.Status(fiber.StatusForbidden).JSON(response.CommonResponse{
				Status:  false,
				Message: "User account is banned",
			})
		}

		var pgID pgtype.UUID
		err := pgID.Scan(claims.UId)
		if err != nil {
			return unauthorized()
		}
		tokenVersion, err := userRepo.GetTokenVersion(c.Context(), pgID)
		if err != nil {
			return unauthorized()
		}

		if tokenVersion != claims.TokenVersion {
			return c.Status(fiber.StatusUnauthorized).JSON(response.CommonResponse{
				Status:  false,
				Message: "Token has been invalidated",
			})
		}

		c.Locals("uid", claims.UId)
		c.Locals("user_claims", claims)

		return c.Next()
	}
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
