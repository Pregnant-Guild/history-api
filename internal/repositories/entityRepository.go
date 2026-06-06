package repositories

import (
	"context"
	json "history-api/pkg/jsonx"

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
	GetEntityIDsByWikiIDs(ctx context.Context, wikiIDs []string) (map[string][]string, error)
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
	return cache.QueryKey(prefix, params)
}

func (r *entityRepository) getByIDsWithFallback(ctx context.Context, ids []string) ([]*models.EntityEntity, error) {
	if len(ids) == 0 {
		return []*models.EntityEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.Key("entity:id", id)
	}
	raws := r.c.MGet(ctx, keys...)

	entities := make([]*models.EntityEntity, 0, len(ids))
	missingToCache := make(map[string]any, len(ids))

	missingPgIds := make([]pgtype.UUID, 0, len(ids))
	for i, b := range raws {
		if len(b) == 0 {
			pgId := pgtype.UUID{}
			err := pgId.Scan(ids[i])
			if err == nil {
				missingPgIds = append(missingPgIds, pgId)
			}
		}
	}

	dbMap := make(map[string]*models.EntityEntity, len(missingPgIds))
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
	cacheId := cache.Key("entity:id", convert.UUIDToString(id))
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
	err := r.c.Get(ctx, queryKey, &cachedIDs)
	if err == nil {
		if len(cachedIDs) == 0 {
			return []*models.EntityEntity{}, nil
		}
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.SearchEntities(ctx, params)
	if err != nil {
		return nil, err
	}
	entities := make([]*models.EntityEntity, 0, len(rows))
	ids := make([]string, 0, len(rows))
	entityToCache := make(map[string]any, len(rows))

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
		entityToCache[cache.Key("entity:id", entity.ID)] = entity
	}

	if len(entityToCache) > 0 {
		_ = r.c.MSet(ctx, entityToCache, constants.NormalCacheDuration)
	}

	_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)

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
	_ = r.c.Del(ctx, cache.Key("entity:id", entity.ID), cache.Key("entity:slug", entity.Slug))
	return &entity, nil
}

func (r *entityRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	err := r.q.DeleteEntity(ctx, id)
	if err != nil {
		return err
	}
	_ = r.c.Del(ctx, cache.Key("entity:id", convert.UUIDToString(id)))
	return nil
}

func (r *entityRepository) GetByProjectID(ctx context.Context, projectID pgtype.UUID) ([]*models.EntityEntity, error) {
	cacheKey := cache.Key("entity:project", convert.UUIDToString(projectID))
	var cachedIDs []string
	err := r.c.Get(ctx, cacheKey, &cachedIDs)
	if err == nil {
		if len(cachedIDs) == 0 {
			return []*models.EntityEntity{}, nil
		}
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.GetEntitiesByProjectId(ctx, projectID)
	if err != nil {
		return nil, err
	}

	entities := make([]*models.EntityEntity, 0, len(rows))
	ids := make([]string, 0, len(rows))
	entityToCache := make(map[string]any, len(rows))

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
		entityToCache[cache.Key("entity:id", entity.ID)] = entity
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
			keys[i] = cache.Key("entity:id", convert.UUIDToString(id))
		}
		_ = r.c.Del(ctx, keys...)
	}
	return nil
}

func (r *entityRepository) GetBySlug(ctx context.Context, slug string) (*models.EntityEntity, error) {
	cacheKey := cache.Key("entity:slug", slug)
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
		keys[i] = cache.Key("entity:slug", slug)
	}
	raws := r.c.MGet(ctx, keys...)

	entities := make([]*models.EntityEntity, 0, len(slugs))
	missingToCache := make(map[string]any, len(slugs))
	missingSlugs := make([]string, 0, len(slugs))

	for i, b := range raws {
		if len(b) == 0 {
			missingSlugs = append(missingSlugs, slugs[i])
		}
	}

	dbMap := make(map[string]*models.EntityEntity, len(missingSlugs))
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
		keys[i] = cache.Key("entity_geometries:geometry", id)
	}

	raws := r.c.MGet(ctx, keys...)
	result := make(map[string][]string, len(geometryIDs))
	missingGeometryIDs := make([]string, 0, len(geometryIDs))
	missingPgIDs := make([]pgtype.UUID, 0, len(geometryIDs))

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

		dbMap := make(map[string][]string, len(missingGeometryIDs))
		for _, id := range missingGeometryIDs {
			dbMap[id] = []string{}
		}
		for _, row := range rows {
			gID := convert.UUIDToString(row.GeometryID)
			eID := convert.UUIDToString(row.EntityID)
			dbMap[gID] = append(dbMap[gID], eID)
		}

		missingToCache := make(map[string]any, len(dbMap))
		for gID, eIDs := range dbMap {
			result[gID] = eIDs
			missingToCache[cache.Key("entity_geometries:geometry", gID)] = eIDs
		}
		if len(missingToCache) > 0 {
			_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
		}
	}

	return result, nil
}

func (r *entityRepository) GetEntityIDsByWikiIDs(ctx context.Context, wikiIDs []string) (map[string][]string, error) {
	if len(wikiIDs) == 0 {
		return make(map[string][]string), nil
	}

	keys := make([]string, len(wikiIDs))
	for i, id := range wikiIDs {
		keys[i] = cache.Key("entity_wikis:wiki", id)
	}

	raws := r.c.MGet(ctx, keys...)
	result := make(map[string][]string, len(wikiIDs))
	missingWikiIDs := make([]string, 0, len(wikiIDs))
	missingPgIDs := make([]pgtype.UUID, 0, len(wikiIDs))

	for i, b := range raws {
		if len(b) > 0 {
			var entityIDs []string
			if err := json.Unmarshal(b, &entityIDs); err == nil {
				result[wikiIDs[i]] = entityIDs
				continue
			}
		}
		missingWikiIDs = append(missingWikiIDs, wikiIDs[i])
		pgID, err := convert.StringToUUID(wikiIDs[i])
		if err == nil {
			missingPgIDs = append(missingPgIDs, pgID)
		}
	}

	if len(missingPgIDs) > 0 {
		rows, err := r.q.GetEntityIDsByWikiIDs(ctx, missingPgIDs)
		if err != nil {
			return nil, err
		}

		dbMap := make(map[string][]string, len(missingWikiIDs))
		for _, id := range missingWikiIDs {
			dbMap[id] = []string{}
		}
		for _, row := range rows {
			wID := convert.UUIDToString(row.WikiID)
			eID := convert.UUIDToString(row.EntityID)
			dbMap[wID] = append(dbMap[wID], eID)
		}

		missingToCache := make(map[string]any, len(dbMap))
		for wID, eIDs := range dbMap {
			result[wID] = eIDs
			missingToCache[cache.Key("entity_wikis:wiki", wID)] = eIDs
		}
		if len(missingToCache) > 0 {
			_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
		}
	}

	return result, nil
}

