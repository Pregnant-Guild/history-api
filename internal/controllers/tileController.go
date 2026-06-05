package controllers

import (
	"fmt"
	"history-api/internal/dtos/response"
	"history-api/internal/services"
	"strconv"

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
	res, err := h.service.GetMetadata(c.Context())
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
	z, x, y, pErr := h.parseTileParams(c)
	if pErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.CommonResponse{
			Status:  false,
			Message: pErr.Error(),
		})
	}

	tile, err := h.service.GetTile(c.Context(), z, x, y)
	if err != nil {
		return c.Status(err.Code).JSON(response.CommonResponse{
			Status:  false,
			Message: err.Message,
		})
	}

	if tile.ContentType != "" {
		c.Set(fiber.HeaderContentType, tile.ContentType)
	}
	if tile.ContentEncoding != "" {
		c.Set(fiber.HeaderContentEncoding, tile.ContentEncoding)
	}
	if tile.CacheControl != "" {
		c.Set(fiber.HeaderCacheControl, tile.CacheControl)
	}

	return c.Status(fiber.StatusOK).Send(tile.Data)
}

func (h *TileController) parseTileParams(c fiber.Ctx) (int, int, int, error) {
	z, err := strconv.Atoi(c.Params("z"))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid z coordinate")
	}

	x, err := strconv.Atoi(c.Params("x"))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid x coordinate")
	}

	y, err := strconv.Atoi(c.Params("y"))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid y coordinate")
	}

	if z < 0 || x < 0 || y < 0 {
		return 0, 0, 0, fmt.Errorf("coordinates must be positive")
	}

	if z > 22 {
		return 0, 0, 0, fmt.Errorf("zoom level too large")
	}

	return z, x, y, nil
}
