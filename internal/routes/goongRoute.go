package routes

import (
	"history-api/internal/controllers"

	"github.com/gofiber/fiber/v3"
)

func GoongRoutes(app *fiber.App, goongController controllers.GoongController) {
	api := app.Group("/proxy")

	api.Get("/*", goongController.Proxy)
}
