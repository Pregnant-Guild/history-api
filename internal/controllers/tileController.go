package controllers

import (
	"context"
	"fmt"
	"history-api/internal/dtos/response"
	"history-api/internal/services"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
)

type TileController struct {
	service services.TileService
}

func NewTileController(svc services.TileService) *TileController {
	return &TileController{service: svc}
}

// GetMetadata godoc
// @Summary Get tile metadata
// @Description Retrieve map metadata
// @Tags Tile
// @Accept json
// @Produce json
// @Success 200 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /tiles/metadata [get]
func (h *TileController) GetMetadata(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := h.service.GetMetadata(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(response.CommonResponse{
		Status: true,
		Data:   res,
	})
}

// GetTile godoc
// @Summary Get a map tile
// @Description Fetch vector or raster map tile data by Z, X, Y coordinates
// @Tags Tile
// @Produce application/octet-stream
// @Param z path int true "Zoom level (0-22)"
// @Param x path int true "X coordinate"
// @Param y path int true "Y coordinate"
// @Success 200 {file} byte
// @Failure 400 {object} response.CommonResponse
// @Failure 500 {object} response.CommonResponse
// @Router /tiles/{z}/{x}/{y} [get]
func (h *TileController) GetTile(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	z, x, y, err := h.parseTileParams(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	data, headers, err := h.service.GetTile(ctx, z, x, y)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Error(),
		})
	}

	for k, v := range headers {
		c.Set(k, v)
	}

	return c.Status(fiber.StatusOK).Send(data)
}

func (h *TileController) parseTileParams(c fiber.Ctx) (int, int, int, error) {
	z, err := strconv.Atoi(c.Params("z"))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid z")
	}

	x, err := strconv.Atoi(c.Params("x"))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid x")
	}

	y, err := strconv.Atoi(c.Params("y"))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid y")
	}

	if z < 0 || x < 0 || y < 0 {
		return 0, 0, 0, fmt.Errorf("coordinates must be positive")
	}

	if z > 22 {
		return 0, 0, 0, fmt.Errorf("zoom level too large")
	}

	return z, x, y, nil
}
