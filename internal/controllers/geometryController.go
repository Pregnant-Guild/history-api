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

type GeometryController struct {
	service services.GeometryService
}

func NewGeometryController(svc services.GeometryService) *GeometryController {
	return &GeometryController{service: svc}
}

// GetGeometryById handles fetching a single geometry by ID.
// @Summary      Get geometry by ID
// @Description  Get detailed information about a specific geometry
// @Tags         Geometries
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Geometry ID"
// @Success      200  {object}  response.CommonResponse
// @Failure      500  {object}  response.CommonResponse
// @Router       /geometries/{id} [get]
func (h *GeometryController) GetGeometryById(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	id := c.Params("id")
	res, err := h.service.GetGeometryByID(ctx, id)
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

// SearchGeometries handles searching for geometries.
// @Summary      Search geometries
// @Description  Search geometries with cursor pagination and spatial filtering
// @Tags         Geometries
// @Accept       json
// @Produce      json
// @Param query query request.SearchGeometryDto false "Search Query"
// @Success      200  {object}  response.CommonResponse
// @Failure      500  {object}  response.CommonResponse
// @Router       /geometries [get]
func (h *GeometryController) SearchGeometries(c fiber.Ctx) error {
	dto := &request.SearchGeometryDto{}
	if err := validator.ValidateQueryDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	res, err := h.service.SearchGeometries(c.Context(), dto)
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

// SearchGeometriesByEntityName handles searching entities by name and returning geometries linked to those entities.
// Cursor pagination is based on entity ID (uuid).
// @Summary      Search geometries by entity name
// @Description  Search entities by name (cursor pagination) and return their linked geometries
// @Tags         Geometries
// @Accept       json
// @Produce      json
// @Param query query request.SearchGeometriesByEntityNameDto false "Search Query"
// @Success      200  {object}  response.CommonResponse
// @Failure      500  {object}  response.CommonResponse
// @Router       /geometries/entity [get]
func (h *GeometryController) SearchGeometriesByEntityName(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dto := &request.SearchGeometriesByEntityNameDto{}
	if err := validator.ValidateQueryDto(c, dto); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status: false,
			Errors: err,
		})
	}

	res, err := h.service.SearchGeometriesByEntityName(ctx, dto)
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

// GetGeometriesByBoundWith handles fetching geometries by their bound_with reference.
// @Summary      Get geometries by bound_with ID
// @Description  Get a list of geometries that are bound to the specified geometry ID
// @Tags         Geometries
// @Accept       json
// @Produce      json
// @Param        bound_with   path      string  true  "Bound-with Geometry ID"
// @Success      200  {object}  response.CommonResponse
// @Failure      500  {object}  response.CommonResponse
// @Router       /geometries/bound-with/{bound_with} [get]
func (h *GeometryController) GetGeometriesByBoundWith(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	boundWith := c.Params("bound_with")
	res, err := h.service.GetGeometriesByBoundWith(ctx, boundWith)
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

