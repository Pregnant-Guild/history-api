package routes

import (
	"history-api/internal/controllers"

	"github.com/gofiber/fiber/v3"
)

func GoongRoutes(app *fiber.App, goongController controllers.GoongController) {
	app.Get("/api/proxy/*", goongController.Proxy)
	app.Get("/map/proxy/*", goongController.Proxy)
}
