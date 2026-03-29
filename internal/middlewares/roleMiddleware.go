package middlewares

import (
	"history-api/internal/dtos/response"
	"history-api/pkg/constants"
	"slices"

	"github.com/gofiber/fiber/v3"
)

func getRoles(c fiber.Ctx) ([]constants.Role, error) {
	claimsVal := c.Locals("user_claims")
	if claimsVal == nil {
		return nil, fiber.ErrUnauthorized
	}

	claims, ok := claimsVal.(*response.JWTClaims)
	if !ok {
		return nil, fiber.ErrUnauthorized
	}

	return claims.Roles, nil
}

func RequireAnyRole(required ...constants.Role) fiber.Handler {
	return func(c fiber.Ctx) error {
		userRoles, err := getRoles(c)
		if err != nil {
			return err
		}

		if len(userRoles) == 0 {
			return fiber.ErrForbidden
		}

		for _, ur := range userRoles {
			if slices.Contains(required, ur) {
				return c.Next()
			}
		}

		return fiber.ErrForbidden
	}
}

func RequireAllRoles(required ...constants.Role) fiber.Handler {
	return func(c fiber.Ctx) error {
		userRoles, err := getRoles(c)
		if err != nil {
			return err
		}

		for _, rr := range required {
			found := slices.Contains(userRoles, rr)
			if !found {
				return fiber.ErrForbidden
			}
		}

		return c.Next()
	}
}
