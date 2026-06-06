package services

import (
	"context"
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/models"
	"history-api/internal/repositories"

	"github.com/gofiber/fiber/v3"
)

type RelationService interface {
	GetRelations(ctx context.Context, req *request.GetRelationsDto) (interface{}, *fiber.Error)
}

type relationService struct {
	wikiRepo     repositories.WikiRepository
	entityRepo   repositories.EntityRepository
	geometryRepo repositories.GeometryRepository
}

func NewRelationService(
	wikiRepo repositories.WikiRepository,
	entityRepo repositories.EntityRepository,
	geometryRepo repositories.GeometryRepository,
) RelationService {
	return &relationService{
		wikiRepo:     wikiRepo,
		entityRepo:   entityRepo,
		geometryRepo: geometryRepo,
	}
}

func (s *relationService) GetRelations(ctx context.Context, req *request.GetRelationsDto) (interface{}, *fiber.Error) {
	switch req.Type {
	case "wiki-entity":
		return s.getEntitiesByWikiIDs(ctx, req.IDs)
	case "entity-wiki":
		return s.getWikisByEntityIDs(ctx, req.IDs)
	case "geometry-entity":
		return s.getEntitiesByGeometryIDs(ctx, req.IDs)
	case "entity-geometry":
		return s.getGeometriesByEntityIDs(ctx, req.IDs)
	case "entity-geometry-child":
		return s.getGeometriesByEntityIDsFiltered(ctx, req.IDs, true)
	case "entity-geometry-alone":
		return s.getGeometriesByEntityIDsFiltered(ctx, req.IDs, false)
	default:
		return nil, fiber.NewError(fiber.StatusBadRequest, "Unsupported relation type")
	}
}

func (s *relationService) getEntitiesByWikiIDs(ctx context.Context, wikiIDs []string) (map[string][]*response.EntityResponse, *fiber.Error) {
	mapping, err := s.entityRepo.GetEntityIDsByWikiIDs(ctx, wikiIDs)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch entity IDs by wiki IDs")
	}

	totalEntityIDs := 0
	for _, eIDs := range mapping {
		totalEntityIDs += len(eIDs)
	}

	entityIDMap := make(map[string]struct{}, totalEntityIDs)
	allEntityIDs := make([]string, 0, totalEntityIDs)
	for _, eIDs := range mapping {
		for _, eID := range eIDs {
			if _, ok := entityIDMap[eID]; !ok {
				entityIDMap[eID] = struct{}{}
				allEntityIDs = append(allEntityIDs, eID)
			}
		}
	}

	entities, err := s.entityRepo.GetByIDs(ctx, allEntityIDs)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch entities")
	}

	entitiesByID := make(map[string]*models.EntityEntity, len(entities))
	for _, e := range entities {
		entitiesByID[e.ID] = e
	}

	result := make(map[string][]*response.EntityResponse, len(wikiIDs))
	for _, idStr := range wikiIDs {
		eIDs, exists := mapping[idStr]
		result[idStr] = make([]*response.EntityResponse, 0, len(eIDs))
		if exists {
			for _, eID := range eIDs {
				if e, found := entitiesByID[eID]; found {
					result[idStr] = append(result[idStr], e.ToResponse())
				}
			}
		}
	}

	return result, nil
}

func (s *relationService) getWikisByEntityIDs(ctx context.Context, entityIDs []string) (map[string][]*response.WikiResponse, *fiber.Error) {
	mapping, err := s.wikiRepo.GetWikiIDsByEntityIDs(ctx, entityIDs)
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

	result := make(map[string][]*response.WikiResponse, len(entityIDs))
	for _, idStr := range entityIDs {
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

func (s *relationService) getEntitiesByGeometryIDs(ctx context.Context, geometryIDs []string) (map[string][]*response.EntityResponse, *fiber.Error) {
	mapping, err := s.entityRepo.GetEntityIDsByGeometryIDs(ctx, geometryIDs)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch entity IDs by geometry IDs")
	}

	totalEntityIDs := 0
	for _, eIDs := range mapping {
		totalEntityIDs += len(eIDs)
	}

	entityIDMap := make(map[string]struct{}, totalEntityIDs)
	allEntityIDs := make([]string, 0, totalEntityIDs)
	for _, eIDs := range mapping {
		for _, eID := range eIDs {
			if _, ok := entityIDMap[eID]; !ok {
				entityIDMap[eID] = struct{}{}
				allEntityIDs = append(allEntityIDs, eID)
			}
		}
	}

	entities, err := s.entityRepo.GetByIDs(ctx, allEntityIDs)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch entities")
	}

	entitiesByID := make(map[string]*models.EntityEntity, len(entities))
	for _, e := range entities {
		entitiesByID[e.ID] = e
	}

	result := make(map[string][]*response.EntityResponse, len(geometryIDs))
	for _, idStr := range geometryIDs {
		eIDs, exists := mapping[idStr]
		result[idStr] = make([]*response.EntityResponse, 0, len(eIDs))
		if exists {
			for _, eID := range eIDs {
				if e, found := entitiesByID[eID]; found {
					result[idStr] = append(result[idStr], e.ToResponse())
				}
			}
		}
	}

	return result, nil
}

func (s *relationService) getGeometriesByEntityIDs(ctx context.Context, entityIDs []string) (map[string][]*response.GeometryResponse, *fiber.Error) {
	mapping, err := s.geometryRepo.GetGeometryIDsByEntityIDs(ctx, entityIDs)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch geometry IDs by entity IDs")
	}

	totalGeometryIDs := 0
	for _, gIDs := range mapping {
		totalGeometryIDs += len(gIDs)
	}

	geometryIDMap := make(map[string]struct{}, totalGeometryIDs)
	allGeometryIDs := make([]string, 0, totalGeometryIDs)
	for _, gIDs := range mapping {
		for _, gID := range gIDs {
			if _, ok := geometryIDMap[gID]; !ok {
				geometryIDMap[gID] = struct{}{}
				allGeometryIDs = append(allGeometryIDs, gID)
			}
		}
	}

	geometries, err := s.geometryRepo.GetByIDs(ctx, allGeometryIDs)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch geometries")
	}

	geometriesByID := make(map[string]*models.GeometryEntity, len(geometries))
	for _, g := range geometries {
		geometriesByID[g.ID] = g
	}

	result := make(map[string][]*response.GeometryResponse, len(entityIDs))
	for _, idStr := range entityIDs {
		gIDs, exists := mapping[idStr]
		result[idStr] = make([]*response.GeometryResponse, 0, len(gIDs))
		if exists {
			for _, gID := range gIDs {
				if g, found := geometriesByID[gID]; found {
					result[idStr] = append(result[idStr], g.ToResponse())
				}
			}
		}
	}

	return result, nil
}

func (s *relationService) getGeometriesByEntityIDsFiltered(ctx context.Context, entityIDs []string, keepBoundWith bool) (map[string][]*response.GeometryResponse, *fiber.Error) {
	mapping, err := s.geometryRepo.GetGeometryIDsByEntityIDs(ctx, entityIDs)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch geometry IDs by entity IDs")
	}

	totalGeometryIDs := 0
	for _, gIDs := range mapping {
		totalGeometryIDs += len(gIDs)
	}

	geometryIDMap := make(map[string]struct{}, totalGeometryIDs)
	allGeometryIDs := make([]string, 0, totalGeometryIDs)
	for _, gIDs := range mapping {
		for _, gID := range gIDs {
			if _, ok := geometryIDMap[gID]; !ok {
				geometryIDMap[gID] = struct{}{}
				allGeometryIDs = append(allGeometryIDs, gID)
			}
		}
	}

	geometries, err := s.geometryRepo.GetByIDs(ctx, allGeometryIDs)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch geometries")
	}

	geometriesByID := make(map[string]*models.GeometryEntity, len(geometries))
	for _, g := range geometries {
		geometriesByID[g.ID] = g
	}

	result := make(map[string][]*response.GeometryResponse, len(entityIDs))
	for _, idStr := range entityIDs {
		gIDs, exists := mapping[idStr]
		result[idStr] = make([]*response.GeometryResponse, 0, len(gIDs))
		if exists {
			for _, gID := range gIDs {
				if g, found := geometriesByID[gID]; found {
					hasBoundWith := g.BoundWith != nil && *g.BoundWith != ""
					if (keepBoundWith && hasBoundWith) || (!keepBoundWith && !hasBoundWith) {
						result[idStr] = append(result[idStr], g.ToResponse())
					}
				}
			}
		}
	}

	return result, nil
}

