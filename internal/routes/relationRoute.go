package routes

import (
	"history-api/internal/controllers"

	"github.com/gofiber/fiber/v3"
)

func RelationRoutes(router fiber.Router, wikiController *controllers.WikiController, entityController *controllers.EntityController) {
	relation := router.Group("/relations")
	relation.Get("/wikis-by-entities", wikiController.GetWikisByEntityIDs)
	relation.Get("/entities-by-geometries", entityController.GetEntitiesByGeometryIDs)
	relation.Get("/wiki-contents/preview", wikiController.GetWikiContentsPreviewByIDs)
}
