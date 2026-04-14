package routes

import (
	"history-api/internal/controllers"

	"github.com/gofiber/fiber/v3"
)

func RasterTileRoutes(app *fiber.App, controller *controllers.RasterTileController) {
	route := app.Group("/raster-tiles")
	route.Get("/metadata", controller.GetMetadata)
	route.Get("/:z/:x/:y", controller.GetTile)
}
