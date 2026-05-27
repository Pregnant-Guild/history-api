package repositories

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/pkg/cache"
	"history-api/pkg/constants"
	"history-api/pkg/convert"
)

type EntityRepository interface {
	GetByID(ctx context.Context, id pgtype.UUID) (*models.EntityEntity, error)
	GetByIDs(ctx context.Context, ids []string) ([]*models.EntityEntity, error)
	GetBySlug(ctx context.Context, slug string) (*models.EntityEntity, error)
	GetBySlugs(ctx context.Context, slugs []string) ([]*models.EntityEntity, error)
	Search(ctx context.Context, params sqlc.SearchEntitiesParams) ([]*models.EntityEntity, error)
	Create(ctx context.Context, params sqlc.CreateEntityParams) (*models.EntityEntity, error)
	Update(ctx context.Context, params sqlc.UpdateEntityParams) (*models.EntityEntity, error)
	Delete(ctx context.Context, id pgtype.UUID) error
	DeleteByIDs(ctx context.Context, ids []pgtype.UUID) error
	GetByProjectID(ctx context.Context, projectID pgtype.UUID) ([]*models.EntityEntity, error)
	GetEntityIDsByGeometryIDs(ctx context.Context, geometryIDs []string) (map[string][]string, error)
	WithTx(tx pgx.Tx) EntityRepository
}

type entityRepository struct {
	q *sqlc.Queries
	c cache.Cache
}

func NewEntityRepository(db sqlc.DBTX, c cache.Cache) EntityRepository {
	return &entityRepository{
		q: sqlc.New(db),
		c: c,
	}
}

func (r *entityRepository) WithTx(tx pgx.Tx) EntityRepository {
	return &entityRepository{
		q: r.q.WithTx(tx),
		c: r.c,
	}
}

func (r *entityRepository) generateQueryKey(prefix string, params any) string {
	b, _ := json.Marshal(params)
	hash := fmt.Sprintf("%x", md5.Sum(b))
	return fmt.Sprintf("%s:%s", prefix, hash)
}

func (r *entityRepository) getByIDsWithFallback(ctx context.Context, ids []string) ([]*models.EntityEntity, error) {
	if len(ids) == 0 {
		return []*models.EntityEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("entity:id:%s", id)
	}
	raws := r.c.MGet(ctx, keys...)

	var entities []*models.EntityEntity
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

	dbMap := make(map[string]*models.EntityEntity)
	if len(missingPgIds) > 0 {
		dbRows, err := r.q.GetEntitiesByIDs(ctx, missingPgIds)
		if err == nil {
			for _, row := range dbRows {
				item := models.EntityEntity{
					ID:          convert.UUIDToString(row.ID),
					Name:        row.Name,
					Slug:        convert.TextToString(row.Slug),
					Description: convert.TextToString(row.Description),
					ProjectID:   convert.UUIDToString(row.ProjectID),
					Status:      convert.Int2ToInt16Ptr(row.Status),
					TimeStart:   convert.Int4ToPtr(row.TimeStart),
					TimeEnd:     convert.Int4ToPtr(row.TimeEnd),
					IsDeleted:   row.IsDeleted,
					CreatedAt:   convert.TimeToPtr(row.CreatedAt),
					UpdatedAt:   convert.TimeToPtr(row.UpdatedAt),
				}
				dbMap[item.ID] = &item
			}
		}
	}

	for i, b := range raws {
		if len(b) > 0 {
			var u models.EntityEntity
			if err := json.Unmarshal(b, &u); err == nil {
				entities = append(entities, &u)
			}
		} else {
			if item, ok := dbMap[ids[i]]; ok {
				entities = append(entities, item)
				missingToCache[keys[i]] = item
			}
		}
	}

	if len(missingToCache) > 0 {
		_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
	}

	return entities, nil
}

func (r *entityRepository) GetByIDs(ctx context.Context, ids []string) ([]*models.EntityEntity, error) {
	return r.getByIDsWithFallback(ctx, ids)
}

func (r *entityRepository) GetByID(ctx context.Context, id pgtype.UUID) (*models.EntityEntity, error) {
	cacheId := fmt.Sprintf("entity:id:%s", convert.UUIDToString(id))
	var entity models.EntityEntity
	err := r.c.Get(ctx, cacheId, &entity)
	if err == nil {
		_ = r.c.Set(ctx, cacheId, entity, constants.NormalCacheDuration)
		return &entity, nil
	}

	row, err := r.q.GetEntityById(ctx, id)
	if err != nil {
		return nil, err
	}

	entity = models.EntityEntity{
		ID:          convert.UUIDToString(row.ID),
		Name:        row.Name,
		Slug:        convert.TextToString(row.Slug),
		Description: convert.TextToString(row.Description),
		ProjectID:   convert.UUIDToString(row.ProjectID),
		Status:      convert.Int2ToInt16Ptr(row.Status),
		TimeStart:   convert.Int4ToPtr(row.TimeStart),
		TimeEnd:     convert.Int4ToPtr(row.TimeEnd),
		IsDeleted:   row.IsDeleted,
		CreatedAt:   convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:   convert.TimeToPtr(row.UpdatedAt),
	}
	_ = r.c.Set(ctx, cacheId, entity, constants.NormalCacheDuration)

	return &entity, nil
}

func (r *entityRepository) Search(ctx context.Context, params sqlc.SearchEntitiesParams) ([]*models.EntityEntity, error) {
	queryKey := r.generateQueryKey("entity:search", params)
	var cachedIDs []string
	if err := r.c.Get(ctx, queryKey, &cachedIDs); err == nil && len(cachedIDs) > 0 {
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.SearchEntities(ctx, params)
	if err != nil {
		return nil, err
	}
	var entities []*models.EntityEntity
	var ids []string
	entityToCache := make(map[string]any)

	for _, row := range rows {
		entity := &models.EntityEntity{
			ID:          convert.UUIDToString(row.ID),
			Name:        row.Name,
			Slug:        convert.TextToString(row.Slug),
			Description: convert.TextToString(row.Description),
			ProjectID:   convert.UUIDToString(row.ProjectID),
			Status:      convert.Int2ToInt16Ptr(row.Status),
			TimeStart:   convert.Int4ToPtr(row.TimeStart),
			TimeEnd:     convert.Int4ToPtr(row.TimeEnd),
			IsDeleted:   row.IsDeleted,
			CreatedAt:   convert.TimeToPtr(row.CreatedAt),
			UpdatedAt:   convert.TimeToPtr(row.UpdatedAt),
		}
		ids = append(ids, entity.ID)
		entities = append(entities, entity)
		entityToCache[fmt.Sprintf("entity:id:%s", entity.ID)] = entity
	}

	if len(entityToCache) > 0 {
		_ = r.c.MSet(ctx, entityToCache, constants.NormalCacheDuration)
	}

	if len(ids) > 0 {
		_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)
	}

	return entities, nil
}

func (r *entityRepository) Create(ctx context.Context, params sqlc.CreateEntityParams) (*models.EntityEntity, error) {
	row, err := r.q.CreateEntity(ctx, params)
	if err != nil {
		return nil, err
	}

	entity := models.EntityEntity{
		ID:          convert.UUIDToString(row.ID),
		Name:        row.Name,
		Slug:        convert.TextToString(row.Slug),
		Description: convert.TextToString(row.Description),
		ProjectID:   convert.UUIDToString(row.ProjectID),
		Status:      convert.Int2ToInt16Ptr(row.Status),
		TimeStart:   convert.Int4ToPtr(row.TimeStart),
		TimeEnd:     convert.Int4ToPtr(row.TimeEnd),
		IsDeleted:   row.IsDeleted,
		CreatedAt:   convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:   convert.TimeToPtr(row.UpdatedAt),
	}

	return &entity, nil
}

func (r *entityRepository) Update(ctx context.Context, params sqlc.UpdateEntityParams) (*models.EntityEntity, error) {
	row, err := r.q.UpdateEntity(ctx, params)
	if err != nil {
		return nil, err
	}
	entity := models.EntityEntity{
		ID:          convert.UUIDToString(row.ID),
		Name:        row.Name,
		Slug:        convert.TextToString(row.Slug),
		Description: convert.TextToString(row.Description),
		ProjectID:   convert.UUIDToString(row.ProjectID),
		Status:      convert.Int2ToInt16Ptr(row.Status),
		TimeStart:   convert.Int4ToPtr(row.TimeStart),
		TimeEnd:     convert.Int4ToPtr(row.TimeEnd),
		IsDeleted:   row.IsDeleted,
		CreatedAt:   convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:   convert.TimeToPtr(row.UpdatedAt),
	}
	_ = r.c.Del(ctx, fmt.Sprintf("entity:id:%s", entity.ID))
	_ = r.c.Del(ctx, fmt.Sprintf("entity:slug:%s", entity.Slug))
	return &entity, nil
}

func (r *entityRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	err := r.q.DeleteEntity(ctx, id)
	if err != nil {
		return err
	}
	_ = r.c.Del(ctx, fmt.Sprintf("entity:id:%s", convert.UUIDToString(id)))
	return nil
}

func (r *entityRepository) GetByProjectID(ctx context.Context, projectID pgtype.UUID) ([]*models.EntityEntity, error) {
	cacheKey := fmt.Sprintf("entity:project:%s", convert.UUIDToString(projectID))
	var cachedIDs []string
	if err := r.c.Get(ctx, cacheKey, &cachedIDs); err == nil && len(cachedIDs) > 0 {
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.GetEntitiesByProjectId(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var entities []*models.EntityEntity
	var ids []string
	entityToCache := make(map[string]any)

	for _, row := range rows {
		entity := &models.EntityEntity{
			ID:          convert.UUIDToString(row.ID),
			Name:        row.Name,
			Slug:        convert.TextToString(row.Slug),
			Description: convert.TextToString(row.Description),
			ProjectID:   convert.UUIDToString(row.ProjectID),
			Status:      convert.Int2ToInt16Ptr(row.Status),
			TimeStart:   convert.Int4ToPtr(row.TimeStart),
			TimeEnd:     convert.Int4ToPtr(row.TimeEnd),
			IsDeleted:   row.IsDeleted,
			CreatedAt:   convert.TimeToPtr(row.CreatedAt),
			UpdatedAt:   convert.TimeToPtr(row.UpdatedAt),
		}
		ids = append(ids, entity.ID)
		entities = append(entities, entity)
		entityToCache[fmt.Sprintf("entity:id:%s", entity.ID)] = entity
	}

	if len(entityToCache) > 0 {
		_ = r.c.MSet(ctx, entityToCache, constants.NormalCacheDuration)
	}
	_ = r.c.Set(ctx, cacheKey, ids, constants.ListCacheDuration)

	return entities, nil
}

func (r *entityRepository) DeleteByIDs(ctx context.Context, ids []pgtype.UUID) error {
	err := r.q.DeleteEntitiesByIDs(ctx, ids)
	if err != nil {
		return err
	}
	if len(ids) > 0 {
		keys := make([]string, len(ids))
		for i, id := range ids {
			keys[i] = fmt.Sprintf("entity:id:%s", convert.UUIDToString(id))
		}
		_ = r.c.Del(ctx, keys...)
	}
	return nil
}

func (r *entityRepository) GetBySlug(ctx context.Context, slug string) (*models.EntityEntity, error) {
	cacheKey := fmt.Sprintf("entity:slug:%s", slug)
	var entity models.EntityEntity
	err := r.c.Get(ctx, cacheKey, &entity)
	if err == nil {
		_ = r.c.Set(ctx, cacheKey, entity, constants.NormalCacheDuration)
		return &entity, nil
	}

	row, err := r.q.GetEntityBySlug(ctx, convert.StringToText(slug))
	if err != nil {
		return nil, err
	}

	entity = models.EntityEntity{
		ID:          convert.UUIDToString(row.ID),
		Name:        row.Name,
		Slug:        convert.TextToString(row.Slug),
		Description: convert.TextToString(row.Description),
		ProjectID:   convert.UUIDToString(row.ProjectID),
		Status:      convert.Int2ToInt16Ptr(row.Status),
		TimeStart:   convert.Int4ToPtr(row.TimeStart),
		TimeEnd:     convert.Int4ToPtr(row.TimeEnd),
		IsDeleted:   row.IsDeleted,
		CreatedAt:   convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:   convert.TimeToPtr(row.UpdatedAt),
	}
	_ = r.c.Set(ctx, cacheKey, entity, constants.NormalCacheDuration)

	return &entity, nil
}

func (r *entityRepository) GetBySlugs(ctx context.Context, slugs []string) ([]*models.EntityEntity, error) {
	if len(slugs) == 0 {
		return []*models.EntityEntity{}, nil
	}
	keys := make([]string, len(slugs))
	for i, slug := range slugs {
		keys[i] = fmt.Sprintf("entity:slug:%s", slug)
	}
	raws := r.c.MGet(ctx, keys...)

	var entities []*models.EntityEntity
	missingToCache := make(map[string]any)
	var missingSlugs []string

	for i, b := range raws {
		if len(b) == 0 {
			missingSlugs = append(missingSlugs, slugs[i])
		}
	}

	dbMap := make(map[string]*models.EntityEntity)
	if len(missingSlugs) > 0 {
		dbRows, err := r.q.GetEntitiesBySlugs(ctx, missingSlugs)
		if err == nil {
			for _, row := range dbRows {
				item := models.EntityEntity{
					ID:          convert.UUIDToString(row.ID),
					Name:        row.Name,
					Slug:        convert.TextToString(row.Slug),
					Description: convert.TextToString(row.Description),
					ProjectID:   convert.UUIDToString(row.ProjectID),
					Status:      convert.Int2ToInt16Ptr(row.Status),
					TimeStart:   convert.Int4ToPtr(row.TimeStart),
					TimeEnd:     convert.Int4ToPtr(row.TimeEnd),
					IsDeleted:   row.IsDeleted,
					CreatedAt:   convert.TimeToPtr(row.CreatedAt),
					UpdatedAt:   convert.TimeToPtr(row.UpdatedAt),
				}
				dbMap[item.Slug] = &item
			}
		}
	}

	for i, b := range raws {
		if len(b) > 0 {
			var u models.EntityEntity
			if err := json.Unmarshal(b, &u); err == nil {
				entities = append(entities, &u)
			}
		} else {
			if item, ok := dbMap[slugs[i]]; ok {
				entities = append(entities, item)
				missingToCache[keys[i]] = item
			}
		}
	}

	if len(missingToCache) > 0 {
		_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
	}

	return entities, nil
}

func (r *entityRepository) GetEntityIDsByGeometryIDs(ctx context.Context, geometryIDs []string) (map[string][]string, error) {
	if len(geometryIDs) == 0 {
		return make(map[string][]string), nil
	}

	keys := make([]string, len(geometryIDs))
	for i, id := range geometryIDs {
		keys[i] = fmt.Sprintf("entity_geometries:geometry:%s", id)
	}

	raws := r.c.MGet(ctx, keys...)
	result := make(map[string][]string)
	var missingGeometryIDs []string
	var missingPgIDs []pgtype.UUID

	for i, b := range raws {
		if len(b) > 0 {
			var entityIDs []string
			if err := json.Unmarshal(b, &entityIDs); err == nil {
				result[geometryIDs[i]] = entityIDs
				continue
			}
		}
		missingGeometryIDs = append(missingGeometryIDs, geometryIDs[i])
		pgID, err := convert.StringToUUID(geometryIDs[i])
		if err == nil {
			missingPgIDs = append(missingPgIDs, pgID)
		}
	}

	if len(missingPgIDs) > 0 {
		rows, err := r.q.GetEntityIDsByGeometryIDs(ctx, missingPgIDs)
		if err != nil {
			return nil, err
		}

		dbMap := make(map[string][]string)
		for _, id := range missingGeometryIDs {
			dbMap[id] = []string{}
		}
		for _, row := range rows {
			gID := convert.UUIDToString(row.GeometryID)
			eID := convert.UUIDToString(row.EntityID)
			dbMap[gID] = append(dbMap[gID], eID)
		}

		missingToCache := make(map[string]any)
		for gID, eIDs := range dbMap {
			result[gID] = eIDs
			missingToCache[fmt.Sprintf("entity_geometries:geometry:%s", gID)] = eIDs
		}
		if len(missingToCache) > 0 {
			_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
		}
	}

	return result, nil
}
