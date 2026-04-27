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
	"github.com/jackc/pgx/v5/pgtype"
)

type GeometryService interface {
	GetGeometryByID(ctx context.Context, id string) (*response.GeometryResponse, *fiber.Error)
	SearchGeometries(ctx context.Context, req *request.SearchGeometryDto) ([]*response.GeometryResponse, *fiber.Error)
}

type geometryService struct {
	geometryRepo repositories.GeometryRepository
}

func NewGeometryService(geometryRepo repositories.GeometryRepository) GeometryService {
	return &geometryService{
		geometryRepo: geometryRepo,
	}
}

func (s *geometryService) GetGeometryByID(ctx context.Context, id string) (*response.GeometryResponse, *fiber.Error) {
	geometryId, err := convert.StringToUUID(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid geometry ID format")
	}
	geometry, err := s.geometryRepo.GetByID(ctx, geometryId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Geometry not found")
	}

	return geometry.ToResponse(), nil
}

func (s *geometryService) SearchGeometries(ctx context.Context, req *request.SearchGeometryDto) ([]*response.GeometryResponse, *fiber.Error) {
	params := sqlc.SearchGeometriesParams{}

	if req.MinLng != nil && req.MinLat != nil && req.MaxLng != nil && req.MaxLat != nil {
		if *req.MaxLng < *req.MinLng || *req.MaxLat < *req.MinLat {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid bounding box coordinates")
		}
		params.SearchMinLng = pgtype.Float8{Float64: *req.MinLng, Valid: true}
		params.SearchMinLat = pgtype.Float8{Float64: *req.MinLat, Valid: true}
		params.SearchMaxLng = pgtype.Float8{Float64: *req.MaxLng, Valid: true}
		params.SearchMaxLat = pgtype.Float8{Float64: *req.MaxLat, Valid: true}
	} else {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Bounding box coordinates are required")
	}

	if req.TimePoint != nil {
		if *req.TimePoint < 0 {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Time point must be non-negative")
		}
		params.TimePoint = pgtype.Int4{Int32: *req.TimePoint, Valid: true}
	}

	if req.EntityID != nil {
		entityId, err := convert.StringToUUID(*req.EntityID)
		if err == nil {
			params.EntityID = entityId
		}
	}

	geometries, err := s.geometryRepo.Search(ctx, params)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to search geometries")
	}

	return models.GeometriesEntityToResponse(geometries), nil
}
