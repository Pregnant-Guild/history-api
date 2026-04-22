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

type EntityService interface {
	GetEntityByID(ctx context.Context, id string) (*response.EntityResponse, error)
	SearchEntities(ctx context.Context, req *request.SearchEntityDto) ([]*response.EntityResponse, error)
}

type entityService struct {
	entityRepo repositories.EntityRepository
}

func NewEntityService(entityRepo repositories.EntityRepository) EntityService {
	return &entityService{
		entityRepo: entityRepo,
	}
}

func (s *entityService) GetEntityByID(ctx context.Context, id string) (*response.EntityResponse, error) {
	entityId, err := convert.StringToUUID(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	entity, err := s.entityRepo.GetByID(ctx, entityId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Entity not found")
	}

	return entity.ToResponse(), nil
}

func (s *entityService) SearchEntities(ctx context.Context, req *request.SearchEntityDto) ([]*response.EntityResponse, error) {
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
		params.Name = req.Name
	}

	entities, err := s.entityRepo.Search(ctx, params)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return models.EntitiesEntityToResponse(entities), nil
}
