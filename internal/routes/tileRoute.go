package routes

import (
	"history-api/internal/controllers"

	"github.com/gofiber/fiber/v3"
)

func TileRoutes(app *fiber.App, controller *controllers.TileController) {
	route := app.Group("/tiles")
	route.Get("/metadata", controller.GetMetadata)
	route.Get("/:z/:x/:y", controller.GetTile)
}
