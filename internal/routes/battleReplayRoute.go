package routes

import (
	"history-api/internal/controllers"

	"github.com/gofiber/fiber/v3"
)

func BattleReplayRoutes(router fiber.Router, battleReplayController *controllers.BattleReplayController) {
	br := router.Group("/battle-replays")
	br.Get("/geometry/:geometryId", battleReplayController.GetBattleReplaysByGeometryId)
	br.Get("/:id", battleReplayController.GetBattleReplayById)
}
