package controllers

import (
	"context"
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/services"
	"history-api/pkg/validator"
	"time"

	"github.com/gofiber/fiber/v3"
)

type RelationController struct {
	service services.RelationService
}

func NewRelationController(svc services.RelationService) *RelationController {
	return &RelationController{service: svc}
}

// GetRelations handles fetching relationships dynamically.
// @Summary      Get generalized batch relations
// @Description  Get relations by type (wiki-entity, entity-wiki, geometry-entity, entity-geometry) and list of IDs
// @Tags         Relations
// @Accept       json
// @Produce      json
// @Param query query request.GetRelationsDto true "Query Parameters"
// @Success      200  {object}  response.CommonResponse
// @Failure      400  {object}  response.CommonResponse
// @Failure      500  {object}  response.CommonResponse
// @Router       /relations [get]
func (h *RelationController) GetRelations(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.GetRelationsDto{}
	if err := validator.ValidateQueryDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	res, err := h.service.GetRelations(ctx, dto)
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
