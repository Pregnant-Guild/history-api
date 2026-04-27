package services

import (
	"context"
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/internal/repositories"
	"history-api/pkg/convert"

	"github.com/gofiber/fiber/v3"
)

type WikiService interface {
	GetWikiByID(ctx context.Context, id string) (*response.WikiResponse, *fiber.Error)
	SearchWikis(ctx context.Context, req *request.SearchWikiDto) ([]*response.WikiResponse, *fiber.Error)
}

type wikiService struct {
	wikiRepo repositories.WikiRepository
}

func NewWikiService(wikiRepo repositories.WikiRepository) WikiService {
	return &wikiService{
		wikiRepo: wikiRepo,
	}
}

func (s *wikiService) GetWikiByID(ctx context.Context, id string) (*response.WikiResponse, *fiber.Error) {
	wikiId, err := convert.StringToUUID(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid wiki ID format")
	}
	wiki, err := s.wikiRepo.GetByID(ctx, wikiId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Wiki not found")
	}

	return wiki.ToResponse(), nil
}

func (s *wikiService) SearchWikis(ctx context.Context, req *request.SearchWikiDto) ([]*response.WikiResponse, *fiber.Error) {
	limit := int32(25)
	if req.Limit > 0 {
		limit = int32(req.Limit)
	}

	params := sqlc.SearchWikisParams{
		LimitCount: limit,
	}
	if req.Cursor != "" {
		cursor, err := convert.StringToUUID(req.Cursor)
		if err == nil {
			params.CursorID = cursor
		}
	}
	if req.Title != "" {
		params.Title = req.Title
	}
	if req.EntityID != "" {
		entityId, err := convert.StringToUUID(req.EntityID)
		if err == nil {
			params.EntityID = entityId
		}
	}

	wikis, err := s.wikiRepo.Search(ctx, params)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to search wikis")
	}

	return models.WikisEntityToResponse(wikis), nil
}
