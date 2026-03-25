package routes

import (
	"history-api/internal/dtos/response"

	"github.com/gofiber/fiber/v3"
)

func NotFoundRoute(app *fiber.App) {
	app.Use(
		func(c fiber.Ctx) error {
			return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
				Status:  false,
				Message: "sorry, endpoint is not found",
			})
		},
	)
}
