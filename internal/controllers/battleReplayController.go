package controllers

import (
	"context"
	"history-api/internal/dtos/response"
	"history-api/internal/services"
	"time"

	"github.com/gofiber/fiber/v3"
)

type BattleReplayController struct {
	service services.BattleReplayService
}

func NewBattleReplayController(svc services.BattleReplayService) *BattleReplayController {
	return &BattleReplayController{service: svc}
}

// GetBattleReplayById handles fetching a single battle replay by ID.
// @Summary      Get battle replay by ID
// @Description  Get detailed information about a specific battle replay
// @Tags         BattleReplays
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Battle Replay ID"
// @Success      200  {object}  response.CommonResponse
// @Failure      404  {object}  response.CommonResponse
// @Router       /battle-replays/{id} [get]
func (h *BattleReplayController) GetBattleReplayById(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	id := c.Params("id")
	res, err := h.service.GetByID(ctx, id)
	if err != nil {
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

// GetBattleReplaysByGeometryId handles fetching battle replays by geometry ID.
// @Summary      Get battle replays by geometry ID
// @Description  Get all battle replays associated with a specific geometry
// @Tags         BattleReplays
// @Accept       json
// @Produce      json
// @Param        geometryId   path      string  true  "Geometry ID"
// @Success      200  {object}  response.CommonResponse
// @Failure      400  {object}  response.CommonResponse
// @Router       /battle-replays/geometry/{geometryId} [get]
func (h *BattleReplayController) GetBattleReplaysByGeometryId(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	geometryID := c.Params("geometryId")
	res, err := h.service.GetByGeometryID(ctx, geometryID)
	if err != nil {
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}
	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}
