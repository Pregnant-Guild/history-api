package repositories

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"history-api/internal/dtos/response"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/pkg/cache"
	"history-api/pkg/constants"
	"history-api/pkg/convert"
)

type GeometryRepository interface {
	GetByID(ctx context.Context, id pgtype.UUID) (*models.GeometryEntity, error)
	GetByIDs(ctx context.Context, ids []string) ([]*models.GeometryEntity, error)
	Search(ctx context.Context, params sqlc.SearchGeometriesParams) ([]*models.GeometryEntity, error)
	SearchByEntityName(ctx context.Context, params sqlc.SearchGeometriesByEntityNameParams) ([]*models.EntityGeometriesSearchEntity, error)
	Create(ctx context.Context, params sqlc.CreateGeometryParams) (*models.GeometryEntity, error)
	Update(ctx context.Context, params sqlc.UpdateGeometryParams) (*models.GeometryEntity, error)
	Delete(ctx context.Context, id pgtype.UUID) error
	CreateEntityGeometries(ctx context.Context, params sqlc.CreateEntityGeometriesParams) error
	BulkDeleteEntityGeometriesByEntityId(ctx context.Context, entityId pgtype.UUID) error
	GetByProjectID(ctx context.Context, projectID pgtype.UUID) ([]*models.GeometryEntity, error)
	GetGeometriesByBoundWith(ctx context.Context, boundWith pgtype.UUID) ([]*models.GeometryEntity, error)
	DeleteByIDs(ctx context.Context, ids []pgtype.UUID) error
	BulkDeleteEntityGeometriesByGeometryID(ctx context.Context, geometryID pgtype.UUID) error
	DeleteEntityGeometry(ctx context.Context, entityID pgtype.UUID, geometryID pgtype.UUID) error
	DeleteEntityGeometriesByProjectID(ctx context.Context, projectID pgtype.UUID) error
	WithTx(tx pgx.Tx) GeometryRepository
}

type geometryRepository struct {
	q  *sqlc.Queries
	c  cache.Cache
	db sqlc.DBTX
}

func NewGeometryRepository(db sqlc.DBTX, c cache.Cache) GeometryRepository {
	return &geometryRepository{
		q:  sqlc.New(db),
		c:  c,
		db: db,
	}
}

func (r *geometryRepository) WithTx(tx pgx.Tx) GeometryRepository {
	return &geometryRepository{
		q:  r.q.WithTx(tx),
		c:  r.c,
		db: tx,
	}
}

func (r *geometryRepository) generateQueryKey(prefix string, params any) string {
	b, _ := json.Marshal(params)
	hash := fmt.Sprintf("%x", md5.Sum(b))
	return fmt.Sprintf("%s:%s", prefix, hash)
}

func (r *geometryRepository) getByIDsWithFallback(ctx context.Context, ids []string) ([]*models.GeometryEntity, error) {
	if len(ids) == 0 {
		return []*models.GeometryEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("geometry:id:%s", id)
	}
	raws := r.c.MGet(ctx, keys...)

	var geometries []*models.GeometryEntity
	missingToCache := make(map[string]any)

	var missingPgIds []pgtype.UUID
	for i, b := range raws {
		if len(b) == 0 {
			pgId := pgtype.UUID{}
			err := pgId.Scan(ids[i])
			if err == nil {
				missingPgIds = append(missingPgIds, pgId)
			}
		}
	}

	dbMap := make(map[string]*models.GeometryEntity)
	if len(missingPgIds) > 0 {
		dbRows, err := r.q.GetGeometriesByIDs(ctx, missingPgIds)
		if err == nil {
			for _, row := range dbRows {
				item := models.GeometryEntity{
					ID:           convert.UUIDToString(row.ID),
					GeoType:      row.GeoType,
					DrawGeometry: row.DrawGeometry,
					BoundWith:    convert.UUIDToStringPtr(row.BoundWith),
					TimeStart:    convert.Int4ToInt32(row.TimeStart),
					TimeEnd:      convert.Int4ToInt32(row.TimeEnd),
					Bbox: &response.Bbox{
						MinLng: row.MinLng,
						MinLat: row.MinLat,
						MaxLng: row.MaxLng,
						MaxLat: row.MaxLat,
					},
					ProjectID: convert.UUIDToString(row.ProjectID),
					IsDeleted: row.IsDeleted,
					CreatedAt: convert.TimeToPtr(row.CreatedAt),
					UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
				}
				dbMap[item.ID] = &item
			}
		}
	}

	for i, b := range raws {
		if len(b) > 0 {
			var u models.GeometryEntity
			if err := json.Unmarshal(b, &u); err == nil {
				geometries = append(geometries, &u)
			}
		} else {
			if item, ok := dbMap[ids[i]]; ok {
				geometries = append(geometries, item)
				missingToCache[keys[i]] = item
			}
		}
	}

	if len(missingToCache) > 0 {
		_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
	}

	return geometries, nil
}

func (r *geometryRepository) GetByIDs(ctx context.Context, ids []string) ([]*models.GeometryEntity, error) {
	return r.getByIDsWithFallback(ctx, ids)
}

func (r *geometryRepository) GetByID(ctx context.Context, id pgtype.UUID) (*models.GeometryEntity, error) {
	cacheId := fmt.Sprintf("geometry:id:%s", convert.UUIDToString(id))
	var geometry models.GeometryEntity
	err := r.c.Get(ctx, cacheId, &geometry)
	if err == nil {
		_ = r.c.Set(ctx, cacheId, geometry, constants.NormalCacheDuration)
		return &geometry, nil
	}

	row, err := r.q.GetGeometryById(ctx, id)
	if err != nil {
		return nil, err
	}

	geometry = models.GeometryEntity{
		ID:           convert.UUIDToString(row.ID),
		GeoType:      row.GeoType,
		DrawGeometry: row.DrawGeometry,
		BoundWith:    convert.UUIDToStringPtr(row.BoundWith),
		TimeStart:    convert.Int4ToInt32(row.TimeStart),
		TimeEnd:      convert.Int4ToInt32(row.TimeEnd),
		Bbox: &response.Bbox{
			MinLng: row.MinLng,
			MinLat: row.MinLat,
			MaxLng: row.MaxLng,
			MaxLat: row.MaxLat,
		},
		ProjectID: convert.UUIDToString(row.ProjectID),
		IsDeleted: row.IsDeleted,
		CreatedAt: convert.TimeToPtr(row.CreatedAt),
		UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
	}
	_ = r.c.Set(ctx, cacheId, geometry, constants.NormalCacheDuration)

	return &geometry, nil
}

func (r *geometryRepository) Search(ctx context.Context, params sqlc.SearchGeometriesParams) ([]*models.GeometryEntity, error) {
	queryKey := r.generateQueryKey("geometry:search", params)
	var cachedIDs []string
	err := r.c.Get(ctx, queryKey, &cachedIDs)
	if err == nil {
		if len(cachedIDs) == 0 {
			return []*models.GeometryEntity{}, nil
		}
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.SearchGeometries(ctx, params)
	if err != nil {
		return nil, err
	}
	var geometries []*models.GeometryEntity
	var ids []string
	geometryToCache := make(map[string]any)

	for _, row := range rows {
		geometry := &models.GeometryEntity{
			ID:           convert.UUIDToString(row.ID),
			GeoType:      row.GeoType,
			DrawGeometry: row.DrawGeometry,
			BoundWith:    convert.UUIDToStringPtr(row.BoundWith),
			TimeStart:    convert.Int4ToInt32(row.TimeStart),
			TimeEnd:      convert.Int4ToInt32(row.TimeEnd),
			Bbox: &response.Bbox{
				MinLng: row.MinLng,
				MinLat: row.MinLat,
				MaxLng: row.MaxLng,
				MaxLat: row.MaxLat,
			},
			ProjectID: convert.UUIDToString(row.ProjectID),
			IsDeleted: row.IsDeleted,
			CreatedAt: convert.TimeToPtr(row.CreatedAt),
			UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
		}
		ids = append(ids, geometry.ID)
		geometries = append(geometries, geometry)
		geometryToCache[fmt.Sprintf("geometry:id:%s", geometry.ID)] = geometry
	}

	if len(geometryToCache) > 0 {
		_ = r.c.MSet(ctx, geometryToCache, constants.NormalCacheDuration)
	}
	_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)

	return geometries, nil
}

func (r *geometryRepository) Create(ctx context.Context, params sqlc.CreateGeometryParams) (*models.GeometryEntity, error) {
	row, err := r.q.CreateGeometry(ctx, params)
	if err != nil {
		return nil, err
	}

	geometry := models.GeometryEntity{
		ID:           convert.UUIDToString(row.ID),
		GeoType:      row.GeoType,
		DrawGeometry: row.DrawGeometry,
		BoundWith:    convert.UUIDToStringPtr(row.BoundWith),
		TimeStart:    convert.Int4ToInt32(row.TimeStart),
		TimeEnd:      convert.Int4ToInt32(row.TimeEnd),
		Bbox: &response.Bbox{
			MinLng: row.MinLng,
			MinLat: row.MinLat,
			MaxLng: row.MaxLng,
			MaxLat: row.MaxLat,
		},
		ProjectID: convert.UUIDToString(row.ProjectID),
		IsDeleted: row.IsDeleted,
		CreatedAt: convert.TimeToPtr(row.CreatedAt),
		UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
	}

	return &geometry, nil
}

func (r *geometryRepository) Update(ctx context.Context, params sqlc.UpdateGeometryParams) (*models.GeometryEntity, error) {
	row, err := r.q.UpdateGeometry(ctx, params)
	if err != nil {
		return nil, err
	}
	geometry := models.GeometryEntity{
		ID:           convert.UUIDToString(row.ID),
		GeoType:      row.GeoType,
		DrawGeometry: row.DrawGeometry,
		BoundWith:    convert.UUIDToStringPtr(row.BoundWith),
		TimeStart:    convert.Int4ToInt32(row.TimeStart),
		TimeEnd:      convert.Int4ToInt32(row.TimeEnd),
		Bbox: &response.Bbox{
			MinLng: row.MinLng,
			MinLat: row.MinLat,
			MaxLng: row.MaxLng,
			MaxLat: row.MaxLat,
		},
		ProjectID: convert.UUIDToString(row.ProjectID),
		IsDeleted: row.IsDeleted,
		CreatedAt: convert.TimeToPtr(row.CreatedAt),
		UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
	}
	_ = r.c.Del(ctx, fmt.Sprintf("geometry:id:%s", geometry.ID))
	return &geometry, nil
}

func (r *geometryRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	err := r.q.DeleteGeometry(ctx, id)
	if err != nil {
		return err
	}
	_ = r.c.Del(ctx, fmt.Sprintf("geometry:id:%s", convert.UUIDToString(id)))
	return nil
}

func (r *geometryRepository) CreateEntityGeometries(ctx context.Context, params sqlc.CreateEntityGeometriesParams) error {
	return r.q.CreateEntityGeometries(ctx, params)
}

func (r *geometryRepository) DeleteEntityGeometriesByProjectID(ctx context.Context, projectID pgtype.UUID) error {
	return r.q.DeleteEntityGeometriesByProjectID(ctx, projectID)
}

func (r *geometryRepository) BulkDeleteEntityGeometriesByEntityId(ctx context.Context, entityId pgtype.UUID) error {
	_, err := r.q.BulkDeleteEntityGeometriesByEntityId(ctx, entityId)
	if err != nil {
		return err
	}
	return nil
}

func (r *geometryRepository) GetByProjectID(ctx context.Context, projectID pgtype.UUID) ([]*models.GeometryEntity, error) {
	cacheKey := fmt.Sprintf("geometry:project:%s", convert.UUIDToString(projectID))
	var cachedIDs []string
	err := r.c.Get(ctx, cacheKey, &cachedIDs)
	if err == nil {
		if len(cachedIDs) == 0 {
			return []*models.GeometryEntity{}, nil
		}
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.GetGeometriesByProjectId(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var geometries []*models.GeometryEntity
	var ids []string
	geometryToCache := make(map[string]any)

	for _, row := range rows {
		geometry := &models.GeometryEntity{
			ID:           convert.UUIDToString(row.ID),
			GeoType:      row.GeoType,
			DrawGeometry: row.DrawGeometry,
			BoundWith:    convert.UUIDToStringPtr(row.BoundWith),
			TimeStart:    convert.Int4ToInt32(row.TimeStart),
			TimeEnd:      convert.Int4ToInt32(row.TimeEnd),
			Bbox: &response.Bbox{
				MinLng: row.MinLng,
				MinLat: row.MinLat,
				MaxLng: row.MaxLng,
				MaxLat: row.MaxLat,
			},
			ProjectID: convert.UUIDToString(row.ProjectID),
			IsDeleted: row.IsDeleted,
			CreatedAt: convert.TimeToPtr(row.CreatedAt),
			UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
		}
		ids = append(ids, geometry.ID)
		geometries = append(geometries, geometry)
		geometryToCache[fmt.Sprintf("geometry:id:%s", geometry.ID)] = geometry
	}

	if len(geometryToCache) > 0 {
		_ = r.c.MSet(ctx, geometryToCache, constants.NormalCacheDuration)
	}
	_ = r.c.Set(ctx, cacheKey, ids, constants.ListCacheDuration)

	return geometries, nil
}

func (r *geometryRepository) GetGeometriesByBoundWith(ctx context.Context, boundWith pgtype.UUID) ([]*models.GeometryEntity, error) {
	cacheKey := fmt.Sprintf("geometry:bound_with:%s", convert.UUIDToString(boundWith))
	var cachedIDs []string
	err := r.c.Get(ctx, cacheKey, &cachedIDs)
	if err == nil {
		if len(cachedIDs) == 0 {
			return []*models.GeometryEntity{}, nil
		}
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.GetGeometriesByBoundWith(ctx, boundWith)
	if err != nil {
		return nil, err
	}

	var geometries []*models.GeometryEntity
	var ids []string
	geometryToCache := make(map[string]any)

	for _, row := range rows {
		geometry := &models.GeometryEntity{
			ID:           convert.UUIDToString(row.ID),
			GeoType:      row.GeoType,
			DrawGeometry: row.DrawGeometry,
			BoundWith:    convert.UUIDToStringPtr(row.BoundWith),
			TimeStart:    convert.Int4ToInt32(row.TimeStart),
			TimeEnd:      convert.Int4ToInt32(row.TimeEnd),
			Bbox: &response.Bbox{
				MinLng: row.MinLng,
				MinLat: row.MinLat,
				MaxLng: row.MaxLng,
				MaxLat: row.MaxLat,
			},
			ProjectID: convert.UUIDToString(row.ProjectID),
			IsDeleted: row.IsDeleted,
			CreatedAt: convert.TimeToPtr(row.CreatedAt),
			UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
		}
		ids = append(ids, geometry.ID)
		geometries = append(geometries, geometry)
		geometryToCache[fmt.Sprintf("geometry:id:%s", geometry.ID)] = geometry
	}

	if len(geometryToCache) > 0 {
		_ = r.c.MSet(ctx, geometryToCache, constants.NormalCacheDuration)
	}
	_ = r.c.Set(ctx, cacheKey, ids, constants.ListCacheDuration)
	return geometries, nil
}

func (r *geometryRepository) DeleteByIDs(ctx context.Context, ids []pgtype.UUID) error {
	err := r.q.DeleteGeometriesByIDs(ctx, ids)
	if err != nil {
		return err
	}
	if len(ids) > 0 {
		keys := make([]string, len(ids))
		for i, id := range ids {
			keys[i] = fmt.Sprintf("geometry:id:%s", convert.UUIDToString(id))
		}
		_ = r.c.Del(ctx, keys...)
	}
	return nil
}

func (r *geometryRepository) BulkDeleteEntityGeometriesByGeometryID(ctx context.Context, geometryID pgtype.UUID) error {
	return r.q.BulkDeleteEntityGeometriesByGeometryID(ctx, geometryID)
}

func (r *geometryRepository) DeleteEntityGeometry(ctx context.Context, entityID pgtype.UUID, geometryID pgtype.UUID) error {
	return r.q.DeleteEntityGeometry(ctx, sqlc.DeleteEntityGeometryParams{
		EntityID:   entityID,
		GeometryID: geometryID,
	})
}

func (r *geometryRepository) getSearchByIDsWithFallback(ctx context.Context, pairs []string) ([]*models.EntityGeometriesSearchEntity, error) {
	if len(pairs) == 0 {
		return []*models.EntityGeometriesSearchEntity{}, nil
	}
	keys := make([]string, len(pairs))
	for i, pair := range pairs {
		keys[i] = fmt.Sprintf("geometry:search:item:%s", pair)
	}
	raws := r.c.MGet(ctx, keys...)

	var results []*models.EntityGeometriesSearchEntity
	missingToCache := make(map[string]any)

	var missingEntityIds []pgtype.UUID
	var missingGeometryIds []pgtype.UUID

	for i, b := range raws {
		if len(b) == 0 {
			parts := strings.Split(pairs[i], ":")
			if len(parts) == 2 {
				eID := pgtype.UUID{}
				gID := pgtype.UUID{}
				if err := eID.Scan(parts[0]); err == nil {
					if err := gID.Scan(parts[1]); err == nil {
						missingEntityIds = append(missingEntityIds, eID)
						missingGeometryIds = append(missingGeometryIds, gID)
					}
				}
			}
		}
	}

	dbMap := make(map[string]*models.EntityGeometriesSearchEntity)
	if len(missingEntityIds) > 0 {
		dbRows, err := r.q.GetEntityGeometriesByPairs(ctx, sqlc.GetEntityGeometriesByPairsParams{
			EntityIds:   missingEntityIds,
			GeometryIds: missingGeometryIds,
		})
		if err == nil {
			for _, row := range dbRows {
				item := &models.EntityGeometriesSearchEntity{
					EntityID:          convert.UUIDToString(row.EntityID),
					EntityName:        row.EntityName,
					EntityDescription: convert.TextToString(row.EntityDescription),
					GeometryID:        convert.UUIDToString(row.GeometryID),
					GeoType:           row.GeoType,
					DrawGeometry:      row.DrawGeometry,
					BoundWith:         convert.UUIDToStringPtr(row.BoundWith),
					TimeStart:         convert.Int4ToPtr(row.TimeStart),
					TimeEnd:           convert.Int4ToPtr(row.TimeEnd),
				}
				key := fmt.Sprintf("%s:%s", item.EntityID, item.GeometryID)
				dbMap[key] = item
			}
		}
	}

	for i, b := range raws {
		if len(b) > 0 {
			var u models.EntityGeometriesSearchEntity
			if err := json.Unmarshal(b, &u); err == nil {
				results = append(results, &u)
			}
		} else {
			if item, ok := dbMap[pairs[i]]; ok {
				results = append(results, item)
				missingToCache[keys[i]] = item
			}
		}
	}

	if len(missingToCache) > 0 {
		_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
	}

	return results, nil
}

func (r *geometryRepository) SearchByEntityName(ctx context.Context, params sqlc.SearchGeometriesByEntityNameParams) ([]*models.EntityGeometriesSearchEntity, error) {

	queryKey := r.generateQueryKey("geometry:search:entity", params)
	var cachedPairs []string
	err := r.c.Get(ctx, queryKey, &cachedPairs)
	if err == nil {
		if len(cachedPairs) == 0 {
			return []*models.EntityGeometriesSearchEntity{}, nil
		}
		return r.getSearchByIDsWithFallback(ctx, cachedPairs)
	}

	rows, err := r.q.SearchGeometriesByEntityName(ctx, params)
	if err != nil {
		return nil, err
	}
	var geometries []*models.EntityGeometriesSearchEntity
	var pairs []string
	geometryToCache := make(map[string]any)

	for _, row := range rows {
		item := &models.EntityGeometriesSearchEntity{
			EntityID:          convert.UUIDToString(row.EntityID),
			EntityName:        row.EntityName,
			EntityDescription: convert.TextToString(row.EntityDescription),
			GeometryID:        convert.UUIDToString(row.GeometryID),
			GeoType:           convert.Int2ToInt16(row.GeoType),
			DrawGeometry:      row.DrawGeometry,
			BoundWith:         convert.UUIDToStringPtr(row.BoundWith),
			TimeStart:         convert.Int4ToPtr(row.TimeStart),
			TimeEnd:           convert.Int4ToPtr(row.TimeEnd),
		}
		pair := fmt.Sprintf("%s:%s", item.EntityID, item.GeometryID)
		geometries = append(geometries, item)
		pairs = append(pairs, pair)
		geometryToCache[fmt.Sprintf("geometry:search:item:%s", pair)] = item
	}

	if len(geometryToCache) > 0 {
		_ = r.c.MSet(ctx, geometryToCache, constants.NormalCacheDuration)
	}
	_ = r.c.Set(ctx, queryKey, pairs, constants.ListCacheDuration)

	return geometries, nil
}
