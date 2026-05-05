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

type EntityService interface {
	GetEntityByID(ctx context.Context, id string) (*response.EntityResponse, *fiber.Error)
	GetEntityBySlug(ctx context.Context, slug string) (*response.EntityResponse, *fiber.Error)
	IsExistEntitySlug(ctx context.Context, slug string) (bool, *fiber.Error)
	SearchEntities(ctx context.Context, req *request.SearchEntityDto) ([]*response.EntityResponse, *fiber.Error)
}

type entityService struct {
	entityRepo repositories.EntityRepository
}

func NewEntityService(entityRepo repositories.EntityRepository) EntityService {
	return &entityService{
		entityRepo: entityRepo,
	}
}

func (s *entityService) GetEntityByID(ctx context.Context, id string) (*response.EntityResponse, *fiber.Error) {
	entityId, err := convert.StringToUUID(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid entity ID format")
	}
	entity, err := s.entityRepo.GetByID(ctx, entityId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Entity not found")
	}

	return entity.ToResponse(), nil
}

func (s *entityService) GetEntityBySlug(ctx context.Context, slug string) (*response.EntityResponse, *fiber.Error) {
	if slug == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Slug is required")
	}
	entity, err := s.entityRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Entity not found")
	}

	return entity.ToResponse(), nil
}

func (s *entityService) IsExistEntitySlug(ctx context.Context, slug string) (bool, *fiber.Error) {
	if slug == "" {
		return false, fiber.NewError(fiber.StatusBadRequest, "Slug is required")
	}
	entity, err := s.entityRepo.GetBySlug(ctx, slug)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fiber.NewError(fiber.StatusInternalServerError, "Failed to check slug existence")
	}
	return entity != nil, nil
}

func (s *entityService) SearchEntities(ctx context.Context, req *request.SearchEntityDto) ([]*response.EntityResponse, *fiber.Error) {
	limit := int32(25)
	if req.Limit > 0 {
		limit = int32(req.Limit)
	}

	params := sqlc.SearchEntitiesParams{
		LimitCount: limit,
	}
	if req.Cursor != "" {
		cursor, err := convert.StringToUUID(req.Cursor)
		if err == nil {
			params.CursorID = cursor
		}
	}

	if req.Name != "" {
		params.Name = convert.PtrToText(&req.Name)
	}

	if req.ProjectID != nil {
		projectID, err := convert.StringToUUID(*req.ProjectID)
		if err == nil {
			params.ProjectID = projectID
		}
	}

	entities, err := s.entityRepo.Search(ctx, params)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to search entities")
	}

	return models.EntitiesEntityToResponse(entities), nil
}
