package routes

import (
	"history-api/internal/controllers"

	"github.com/gofiber/fiber/v3"
)

func GeometryRoutes(router fiber.Router, geometryController *controllers.GeometryController) {
	geometry := router.Group("/geometries")
	geometry.Get("/", geometryController.SearchGeometries)
	geometry.Get("/:id", geometryController.GetGeometryById)
}
