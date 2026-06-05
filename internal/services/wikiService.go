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
	GetWikisByEntityIDs(ctx context.Context, req *request.GetWikisByEntityIDsDto) (map[string][]*response.WikiResponse, *fiber.Error)
	GetWikiContentsPreviewByIDs(ctx context.Context, req *request.GetWikiContentsPreviewDto) ([]*response.WikiContentPreviewResponse, *fiber.Error)
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

func (s *wikiService) GetWikisByEntityIDs(ctx context.Context, req *request.GetWikisByEntityIDsDto) (map[string][]*response.WikiResponse, *fiber.Error) {
	mapping, err := s.wikiRepo.GetWikiIDsByEntityIDs(ctx, req.EntityIDs)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch wiki IDs by entity IDs")
	}

	totalWikiIDs := 0
	for _, wIDs := range mapping {
		totalWikiIDs += len(wIDs)
	}

	wikiIDMap := make(map[string]struct{}, totalWikiIDs)
	allWikiIDs := make([]string, 0, totalWikiIDs)
	for _, wIDs := range mapping {
		for _, wID := range wIDs {
			if _, ok := wikiIDMap[wID]; !ok {
				wikiIDMap[wID] = struct{}{}
				allWikiIDs = append(allWikiIDs, wID)
			}
		}
	}

	wikis, err := s.wikiRepo.GetByIDs(ctx, allWikiIDs)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch wikis")
	}

	wikisByID := make(map[string]*models.WikiEntity, len(wikis))
	for _, w := range wikis {
		wikisByID[w.ID] = w
	}

	result := make(map[string][]*response.WikiResponse, len(req.EntityIDs))
	for _, idStr := range req.EntityIDs {
		wIDs, exists := mapping[idStr]
		result[idStr] = make([]*response.WikiResponse, 0, len(wIDs))
		if exists {
			for _, wID := range wIDs {
				if w, found := wikisByID[wID]; found {
					result[idStr] = append(result[idStr], w.ToResponse())
				}
			}
		}
	}

	return result, nil
}

func (s *wikiService) GetWikiContentsPreviewByIDs(ctx context.Context, req *request.GetWikiContentsPreviewDto) ([]*response.WikiContentPreviewResponse, *fiber.Error) {
	contents, err := s.wikiRepo.GetContentByIDs(ctx, req.IDs)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch wiki contents")
	}

	results := make([]*response.WikiContentPreviewResponse, 0, len(contents))
	for _, c := range contents {
		results = append(results, &response.WikiContentPreviewResponse{
			ID:        c.ID,
			Preview:   c.Preview,
			CreatedAt: c.CreatedAt,
		})
	}

	return results, nil
}
