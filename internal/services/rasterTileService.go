package services

import (
	"context"
	"history-api/internal/repositories"

	"github.com/gofiber/fiber/v3"
)

type RasterTileService interface {
	GetMetadata(ctx context.Context) (map[string]string, *fiber.Error)
	GetTile(ctx context.Context, z, x, y int) (TileResponse, *fiber.Error)
}

type rasterTileService struct {
	tileRepo repositories.RasterTileRepository
}

func NewRasterTileService(
	TileRepo repositories.RasterTileRepository,
) RasterTileService {
	return &rasterTileService{
		tileRepo: TileRepo,
	}
}

func (t *rasterTileService) GetMetadata(ctx context.Context) (map[string]string, *fiber.Error) {
	metaData, err := t.tileRepo.GetMetadata(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch map metadata")
	}
	return metaData, nil
}

func (t *rasterTileService) GetTile(ctx context.Context, z, x, y int) (TileResponse, *fiber.Error) {
	data, format, err := t.tileRepo.GetTile(ctx, z, x, y)
	if err != nil {
		return TileResponse{}, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch tile data")
	}

	res := TileResponse{
		Data:         data,
		CacheControl: tileCacheControl,
	}
	switch format {
	case "png":
		res.ContentType = "image/png"
	case "jpg", "jpeg":
		res.ContentType = "image/jpeg"
	case "webp":
		res.ContentType = "image/webp"
	default:
		res.ContentType = "application/octet-stream"
	}

	return res, nil
}
