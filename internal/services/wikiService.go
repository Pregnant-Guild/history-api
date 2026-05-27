package services

import (
	"context"
	"database/sql"
	"errors"
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
	GetWikiBySlug(ctx context.Context, slug string) (*response.WikiResponse, *fiber.Error)
	IsExistWikiSlug(ctx context.Context, slug string) (bool, *fiber.Error)
	SearchWikis(ctx context.Context, req *request.SearchWikiDto) ([]*response.WikiResponse, *fiber.Error)
	GetWikiContentByID(ctx context.Context, id string) (*response.WikiContentResponse, *fiber.Error)
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

func (s *wikiService) GetWikiBySlug(ctx context.Context, slug string) (*response.WikiResponse, *fiber.Error) {
	if slug == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Slug is required")
	}
	wiki, err := s.wikiRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Wiki not found")
	}

	return wiki.ToResponse(), nil
}

func (s *wikiService) IsExistWikiSlug(ctx context.Context, slug string) (bool, *fiber.Error) {
	if slug == "" {
		return false, fiber.NewError(fiber.StatusBadRequest, "Slug is required")
	}
	wiki, err := s.wikiRepo.GetBySlug(ctx, slug)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fiber.NewError(fiber.StatusInternalServerError, "Failed to check slug existence")
	}
	return wiki != nil, nil
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

	if req.ProjectID != nil {
		projectID, err := convert.StringToUUID(*req.ProjectID)
		if err == nil {
			params.ProjectID = projectID
		}
	}

	wikis, err := s.wikiRepo.Search(ctx, params)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to search wikis")
	}

	return models.WikisEntityToResponse(wikis), nil
}

func (s *wikiService) GetWikiContentByID(ctx context.Context, id string) (*response.WikiContentResponse, *fiber.Error) {
	contentId, err := convert.StringToUUID(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid content ID format")
	}
	content, err := s.wikiRepo.GetContentByID(ctx, contentId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Wiki content not found")
	}

	return &response.WikiContentResponse{
		ID:        content.ID,
		WikiID:    content.WikiID,
		Title:     content.Title,
		Content:   content.Content,
		Preview:   content.Preview,
		CreatedAt: content.CreatedAt,
	}, nil
}
