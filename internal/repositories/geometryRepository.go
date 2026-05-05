package repositories

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"

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
	SearchByEntityName(ctx context.Context, name string, cursorID *pgtype.UUID, limit int32) ([]EntityGeometriesSearchRow, error)
	Create(ctx context.Context, params sqlc.CreateGeometryParams) (*models.GeometryEntity, error)
	Update(ctx context.Context, params sqlc.UpdateGeometryParams) (*models.GeometryEntity, error)
	Delete(ctx context.Context, id pgtype.UUID) error
	CreateEntityGeometries(ctx context.Context, params sqlc.CreateEntityGeometriesParams) error
	BulkDeleteEntityGeometriesByEntityId(ctx context.Context, entityId pgtype.UUID) error
	GetByProjectID(ctx context.Context, projectID pgtype.UUID) ([]*models.GeometryEntity, error)
	DeleteByIDs(ctx context.Context, ids []pgtype.UUID) error
	BulkDeleteEntityGeometriesByGeometryID(ctx context.Context, geometryID pgtype.UUID) error
	DeleteEntityGeometry(ctx context.Context, entityID pgtype.UUID, geometryID pgtype.UUID) error
	DeleteEntityGeometriesByProjectID(ctx context.Context, projectID pgtype.UUID) error
	WithTx(tx pgx.Tx) GeometryRepository
}

type geometryRepository struct {
	q *sqlc.Queries
	c cache.Cache
	db sqlc.DBTX
}

func NewGeometryRepository(db sqlc.DBTX, c cache.Cache) GeometryRepository {
	return &geometryRepository{
		q: sqlc.New(db),
		c: c,
		db: db,
	}
}

func (r *geometryRepository) WithTx(tx pgx.Tx) GeometryRepository {
	return &geometryRepository{
		q: r.q.WithTx(tx),
		c: r.c,
		db: tx,
	}
}

// EntityGeometriesSearchRow is a "flattened" row shape for searching geometries by entity name.
// FE will display entity list by entity_name, while still having geometries in the same response.
type EntityGeometriesSearchRow struct {
	EntityID          string          `json:"entity_id"`
	EntityName        string          `json:"name"`
	EntityDescription string          `json:"description"`
	GeometryID        string          `json:"id"`
	GeoType           int16           `json:"geo_type"`
	DrawGeometry      json.RawMessage `json:"draw_geometry"`
	Binding           json.RawMessage `json:"binding,omitempty"`
	TimeStart         *int32          `json:"time_start,omitempty"`
	TimeEnd           *int32          `json:"time_end,omitempty"`
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
					Binding:      row.Binding,
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
		Binding:      row.Binding,
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
		Binding:      row.Binding,
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
	if err := r.c.Get(ctx, cacheKey, &cachedIDs); err == nil && len(cachedIDs) > 0 {
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
			Binding:      row.Binding,
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
	if len(ids) > 0 {
		_ = r.c.Set(ctx, cacheKey, ids, constants.ListCacheDuration)
	}

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

func (r *geometryRepository) SearchByEntityName(
	ctx context.Context,
	name string,
	cursorID *pgtype.UUID,
	limit int32,
) ([]EntityGeometriesSearchRow, error) {
	// Cursor is entity_id (not geometry_id) so FE can render entities as the primary list.
	const q = `
WITH matched_entities AS (
    SELECT
        e.id,
        e.name,
        e.description
    FROM entities e
    WHERE e.is_deleted = false
      AND ($1::text = '' OR e.name ILIKE '%' || $1::text || '%')
      AND ($2::uuid IS NULL OR e.id < $2::uuid)
    ORDER BY e.id DESC
    LIMIT $3
)
SELECT
    me.id AS entity_id,
    me.name AS entity_name,
    COALESCE(me.description, '') AS entity_description,
    g.id AS geometry_id,
    g.geo_type,
    g.draw_geometry,
    g.binding,
    g.time_start,
    g.time_end
FROM matched_entities me
LEFT JOIN entity_geometries eg
    ON eg.entity_id = me.id
LEFT JOIN geometries g
    ON g.id = eg.geometry_id
   AND g.is_deleted = false
ORDER BY me.id DESC, g.id DESC
`

	var cursor pgtype.UUID
	if cursorID != nil {
		cursor = *cursorID
	} else {
		cursor = pgtype.UUID{Valid: false}
	}

	rows, err := r.db.Query(ctx, q, name, cursor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]EntityGeometriesSearchRow, 0)
	for rows.Next() {
		var entityID pgtype.UUID
		var entityName pgtype.Text
		var entityDesc pgtype.Text
		var geoID pgtype.UUID
		var geoType pgtype.Int2
		var draw json.RawMessage
		var binding json.RawMessage
		var timeStart pgtype.Int4
		var timeEnd pgtype.Int4

		if err := rows.Scan(
			&entityID,
			&entityName,
			&entityDesc,
			&geoID,
			&geoType,
			&draw,
			&binding,
			&timeStart,
			&timeEnd,
		); err != nil {
			return nil, err
		}

		var ts *int32
		if timeStart.Valid {
			v := timeStart.Int32
			ts = &v
		}
		var te *int32
		if timeEnd.Valid {
			v := timeEnd.Int32
			te = &v
		}

		row := EntityGeometriesSearchRow{
			EntityID:          convert.UUIDToString(entityID),
			EntityName:        entityName.String,
			EntityDescription: entityDesc.String,
			GeometryID:        convert.UUIDToString(geoID),
			GeoType:           geoType.Int16,
			DrawGeometry:      draw,
			Binding:           binding,
			TimeStart:         ts,
			TimeEnd:           te,
		}

		// Entity with no geometry will have NULL geometry_id.
		if !geoID.Valid {
			row.GeometryID = ""
			row.GeoType = 0
			row.DrawGeometry = nil
			row.Binding = nil
			row.TimeStart = nil
			row.TimeEnd = nil
		}

		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
