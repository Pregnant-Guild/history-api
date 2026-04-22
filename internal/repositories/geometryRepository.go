package repositories

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"

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
	Create(ctx context.Context, params sqlc.CreateGeometryParams) (*models.GeometryEntity, error)
	Update(ctx context.Context, params sqlc.UpdateGeometryParams) (*models.GeometryEntity, error)
	Delete(ctx context.Context, id pgtype.UUID) error
	CreateEntityGeometries(ctx context.Context, params sqlc.CreateEntityGeometriesParams) error
	BulkDeleteEntityGeometriesByEntityId(ctx context.Context, entityId pgtype.UUID) error
}

type geometryRepository struct {
	q *sqlc.Queries
	c cache.Cache
}

func NewGeometryRepository(db sqlc.DBTX, c cache.Cache) GeometryRepository {
	return &geometryRepository{
		q: sqlc.New(db),
		c: c,
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

	for i, b := range raws {
		if len(b) > 0 {
			var g models.GeometryEntity
			if err := json.Unmarshal(b, &g); err == nil {
				geometries = append(geometries, &g)
			}
		} else {
			pgId := pgtype.UUID{}
			err := pgId.Scan(ids[i])
			if err != nil {
				continue
			}
			dbGeometry, err := r.GetByID(ctx, pgId)
			if err == nil && dbGeometry != nil {
				geometries = append(geometries, dbGeometry)
				missingToCache[keys[i]] = dbGeometry
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
		Binding:      row.Binding,
		TimeStart:    convert.Int4ToInt32(row.TimeStart),
		TimeEnd:      convert.Int4ToInt32(row.TimeEnd),
		Bbox: &response.Bbox{
			MinLng: row.MinLng,
			MinLat: row.MinLat,
			MaxLng: row.MaxLng,
			MaxLat: row.MaxLat,
		},
		IsDeleted:    row.IsDeleted,
		CreatedAt:    convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:    convert.TimeToPtr(row.UpdatedAt),
	}
	_ = r.c.Set(ctx, cacheId, geometry, constants.NormalCacheDuration)

	return &geometry, nil
}

func (r *geometryRepository) Search(ctx context.Context, params sqlc.SearchGeometriesParams) ([]*models.GeometryEntity, error) {
	queryKey := r.generateQueryKey("geometry:search", params)
	var cachedIDs []string
	if err := r.c.Get(ctx, queryKey, &cachedIDs); err == nil && len(cachedIDs) > 0 {
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
			Binding:      row.Binding,
			TimeStart:    convert.Int4ToInt32(row.TimeStart),
			TimeEnd:      convert.Int4ToInt32(row.TimeEnd),
			Bbox: &response.Bbox{
				MinLng: row.MinLng,
				MinLat: row.MinLat,
				MaxLng: row.MaxLng,
				MaxLat: row.MaxLat,
			},
			IsDeleted:    row.IsDeleted,
			CreatedAt:    convert.TimeToPtr(row.CreatedAt),
			UpdatedAt:    convert.TimeToPtr(row.UpdatedAt),
		}
		ids = append(ids, geometry.ID)
		geometries = append(geometries, geometry)
		geometryToCache[fmt.Sprintf("geometry:id:%s", geometry.ID)] = geometry
	}

	if len(geometryToCache) > 0 {
		_ = r.c.MSet(ctx, geometryToCache, constants.NormalCacheDuration)
	}
	if len(ids) > 0 {
		_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)
	}

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
		Binding:      row.Binding,
		TimeStart:    convert.Int4ToInt32(row.TimeStart),
		TimeEnd:      convert.Int4ToInt32(row.TimeEnd),
		Bbox: &response.Bbox{
			MinLng: row.MinLng,
			MinLat: row.MinLat,
			MaxLng: row.MaxLng,
			MaxLat: row.MaxLat,
		},
		IsDeleted:    row.IsDeleted,
		CreatedAt:    convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:    convert.TimeToPtr(row.UpdatedAt),
	}
	_ = r.c.Set(ctx, fmt.Sprintf("geometry:id:%s", geometry.ID), geometry, constants.NormalCacheDuration)

	go func() {
		bgCtx := context.Background()
		_ = r.c.DelByPattern(bgCtx, "geometry:search*")
	}()
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
		Binding:      row.Binding,
		TimeStart:    convert.Int4ToInt32(row.TimeStart),
		TimeEnd:      convert.Int4ToInt32(row.TimeEnd),
		Bbox: &response.Bbox{
			MinLng: row.MinLng,
			MinLat: row.MinLat,
			MaxLng: row.MaxLng,
			MaxLat: row.MaxLat,
		},
		IsDeleted:    row.IsDeleted,
		CreatedAt:    convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:    convert.TimeToPtr(row.UpdatedAt),
	}
	_ = r.c.Set(ctx, fmt.Sprintf("geometry:id:%s", geometry.ID), geometry, constants.NormalCacheDuration)
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
	err := r.q.CreateEntityGeometries(ctx, params)
	if err != nil {
		return err
	}
	return err
}

func (r *geometryRepository) BulkDeleteEntityGeometriesByEntityId(ctx context.Context, entityId pgtype.UUID) error {
	geometryIDs, err := r.q.BulkDeleteEntityGeometriesByEntityId(ctx, entityId)
	if err != nil {
		return err
	}
	if len(geometryIDs) > 0 {
		keys := make([]string, len(geometryIDs))
		for i, id := range geometryIDs {
			keys[i] = fmt.Sprintf("geometry:id:%s", convert.UUIDToString(id))
		}
		go func() {
			_ = r.c.Del(context.Background(), keys...)
		}()
	}
	return nil
}
