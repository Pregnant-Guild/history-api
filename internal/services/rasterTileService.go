package services

import (
	"context"
	"history-api/internal/repositories"

	"github.com/gofiber/fiber/v3"
)

type RasterTileService interface {
	GetMetadata(ctx context.Context) (map[string]string, error)
	GetTile(ctx context.Context, z, x, y int) ([]byte, map[string]string, error)
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

func (t *rasterTileService) GetMetadata(ctx context.Context) (map[string]string, error) {
	metaData, err := t.tileRepo.GetMetadata(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return metaData, nil
}


func (t *rasterTileService) GetTile(ctx context.Context, z, x, y int) ([]byte, map[string]string, error) {
	contentType := make(map[string]string)

	data, format, err := t.tileRepo.GetTile(ctx, z, x, y)
	if err != nil {
		return nil, contentType, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	
	switch format {
	case "png":
		contentType["Content-Type"] = "image/png"
	case "jpg", "jpeg":
		contentType["Content-Type"] = "image/jpeg"
	case "webp":
		contentType["Content-Type"] = "image/webp"
	default:
		contentType["Content-Type"] = "application/octet-stream"
	}

	return data, contentType, nil

}
