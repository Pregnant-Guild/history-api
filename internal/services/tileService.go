package services

import (
	"context"
	"history-api/internal/repositories"

	"github.com/gofiber/fiber/v3"
)

const tileCacheControl = "public, max-age=31536000, immutable"

type TileResponse struct {
	Data            []byte
	ContentType     string
	ContentEncoding string
	CacheControl    string
}

type TileService interface {
	GetMetadata(ctx context.Context) (map[string]string, *fiber.Error)
	GetTile(ctx context.Context, z, x, y int) (TileResponse, *fiber.Error)
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

func (t *tileService) GetMetadata(ctx context.Context) (map[string]string, *fiber.Error) {
	metaData, err := t.tileRepo.GetMetadata(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch map metadata")
	}
	return metaData, nil
}

func (t *tileService) GetTile(ctx context.Context, z, x, y int) (TileResponse, *fiber.Error) {
	data, format, isPBF, err := t.tileRepo.GetTile(ctx, z, x, y)
	if err != nil {
		return TileResponse{}, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch tile data")
	}

	res := TileResponse{
		Data:         data,
		CacheControl: tileCacheControl,
	}
	switch format {
	case "pbf":
		res.ContentType = "application/x-protobuf"
	case "png":
		res.ContentType = "image/png"
	case "jpg", "jpeg":
		res.ContentType = "image/jpeg"
	default:
		res.ContentType = "application/octet-stream"
	}

	if isPBF {
		res.ContentEncoding = "gzip"
	}

	return res, nil
}
