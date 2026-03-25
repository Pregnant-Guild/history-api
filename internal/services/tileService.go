package services

import (
	"context"
	"history-api/internal/repositories"

	"github.com/gofiber/fiber/v3"
)

type TileService interface {
	GetMetadata(ctx context.Context) (map[string]string, error)
	GetTile(ctx context.Context, z, x, y int) ([]byte, map[string]string, error)
}

type tileService struct {
	tileRepo repositories.TileRepository
}

func NewTileService(
	TileRepo repositories.TileRepository,
) TileService {
	return &tileService{
		tileRepo: TileRepo,
	}
}

func (t *tileService) GetMetadata(ctx context.Context) (map[string]string, error) {
	metaData, err := t.tileRepo.GetMetadata(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return metaData, nil
}


func (t *tileService) GetTile(ctx context.Context, z, x, y int) ([]byte, map[string]string, error) {
	contentType := make(map[string]string)

	data, format, isPBF, err := t.tileRepo.GetTile(ctx, z, x, y)
	if err != nil {
		return nil, contentType, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	contentType["Content-Type"] = "image/png"
	if format == "jpg" {
		contentType["Content-Type"] = "image/jpeg"
	}
	if format == "pbf" {
		contentType["Content-Type"] = "application/x-protobuf"
	}

	if isPBF {
		contentType["Content-Encoding"] = "gzip"
	}

	return data, contentType, nil

}
