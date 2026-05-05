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
	SearchGeometriesByEntityName(ctx context.Context, req *request.SearchGeometriesByEntityNameDto) (*response.SearchGeometriesByEntityNameResponse, *fiber.Error)
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
		params.TimePoint = pgtype.Int4{Int32: *req.TimePoint, Valid: true}
	}

	if req.EntityID != nil {
		entityId, err := convert.StringToUUID(*req.EntityID)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid entity ID format")
		}
		params.EntityID = entityId
	}

	geometries, err := s.geometryRepo.Search(ctx, params)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to search geometries")
	}

	return models.GeometriesEntityToResponse(geometries), nil
}

func (s *geometryService) SearchGeometriesByEntityName(
	ctx context.Context,
	req *request.SearchGeometriesByEntityNameDto,
) (*response.SearchGeometriesByEntityNameResponse, *fiber.Error) {
	limit := int32(req.Limit)
	if limit <= 0 {
		limit = 20
	}

	var cursorID pgtype.UUID
	if req.Cursor != "" {
		id, err := convert.StringToUUID(req.Cursor)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid cursor format")
		}
		cursorID = id
	}

	rows, err := s.geometryRepo.SearchByEntityName(ctx, sqlc.SearchGeometriesByEntityNameParams{
		Name:       convert.StringToText(req.Name),
		CursorID:   cursorID,
		LimitCount: limit,
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to search geometries by entity name")
	}

	byEntity := make(map[string]*response.EntityGeometriesSearchItem)
	order := make([]string, 0)

	for _, row := range rows {
		item := byEntity[row.EntityID]
		if item == nil {
			item = &response.EntityGeometriesSearchItem{
				EntityID:    row.EntityID,
				Name:        row.EntityName,
				Description: row.EntityDescription,
				Geometries:  make([]*response.EntityGeometrySearchGeo, 0),
			}
			byEntity[row.EntityID] = item
			order = append(order, row.EntityID)
		}

		if row.GeometryID == "" || len(row.DrawGeometry) == 0 {
			continue
		}

		item.Geometries = append(item.Geometries, &response.EntityGeometrySearchGeo{
			ID:           row.GeometryID,
			GeoType:      row.GeoType,
			DrawGeometry: row.DrawGeometry,
			Binding:      row.Binding,
			TimeStart:    row.TimeStart,
			TimeEnd:      row.TimeEnd,
		})
	}

	items := make([]*response.EntityGeometriesSearchItem, 0, len(order))
	for _, entityID := range order {
		items = append(items, byEntity[entityID])
	}

	nextCursor := ""
	if len(order) > 0 {
		nextCursor = order[len(order)-1]
	}

	return &response.SearchGeometriesByEntityNameResponse{
		Items:      items,
		NextCursor: nextCursor,
	}, nil
}
