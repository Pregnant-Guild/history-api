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
	Search(ctx context.Context, params sqlc.SearchEntitiesParams) ([]*models.EntityEntity, error)
	Create(ctx context.Context, params sqlc.CreateEntityParams) (*models.EntityEntity, error)
	Update(ctx context.Context, params sqlc.UpdateEntityParams) (*models.EntityEntity, error)
	Delete(ctx context.Context, id pgtype.UUID) error
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
					ID:           convert.UUIDToString(row.ID),
					Name:         row.Name,
					Description:  convert.TextToString(row.Description),
					ThumbnailUrl: convert.TextToString(row.ThumbnailUrl),
					IsDeleted:    row.IsDeleted,
					CreatedAt:    convert.TimeToPtr(row.CreatedAt),
					UpdatedAt:    convert.TimeToPtr(row.UpdatedAt),
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
		ID:           convert.UUIDToString(row.ID),
		Name:         row.Name,
		Description:  convert.TextToString(row.Description),
		ThumbnailUrl: convert.TextToString(row.ThumbnailUrl),
		IsDeleted:    row.IsDeleted,
		CreatedAt:    convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:    convert.TimeToPtr(row.UpdatedAt),
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
			ID:           convert.UUIDToString(row.ID),
			Name:         row.Name,
			Description:  convert.TextToString(row.Description),
			ThumbnailUrl: convert.TextToString(row.ThumbnailUrl),
			IsDeleted:    row.IsDeleted,
			CreatedAt:    convert.TimeToPtr(row.CreatedAt),
			UpdatedAt:    convert.TimeToPtr(row.UpdatedAt),
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
		ID:           convert.UUIDToString(row.ID),
		Name:         row.Name,
		Description:  convert.TextToString(row.Description),
		ThumbnailUrl: convert.TextToString(row.ThumbnailUrl),
		IsDeleted:    row.IsDeleted,
		CreatedAt:    convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:    convert.TimeToPtr(row.UpdatedAt),
	}
	go func() {
		_ = r.c.DelByPattern(context.Background(), "entity:search*")
	}()
	return &entity, nil
}

func (r *entityRepository) Update(ctx context.Context, params sqlc.UpdateEntityParams) (*models.EntityEntity, error) {
	row, err := r.q.UpdateEntity(ctx, params)
	if err != nil {
		return nil, err
	}
	entity := models.EntityEntity{
		ID:           convert.UUIDToString(row.ID),
		Name:         row.Name,
		Description:  convert.TextToString(row.Description),
		ThumbnailUrl: convert.TextToString(row.ThumbnailUrl),
		IsDeleted:    row.IsDeleted,
		CreatedAt:    convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:    convert.TimeToPtr(row.UpdatedAt),
	}
	_ = r.c.Del(ctx, fmt.Sprintf("entity:id:%s", entity.ID))
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
