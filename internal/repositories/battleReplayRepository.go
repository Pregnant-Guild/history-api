package repositories

import (
	"context"
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

type BattleReplayRepository interface {
	GetByID(ctx context.Context, id pgtype.UUID) (*models.BattleReplayEntity, error)
	GetByIDs(ctx context.Context, ids []string) ([]*models.BattleReplayEntity, error)
	GetByGeometryID(ctx context.Context, geometryID pgtype.UUID) ([]*models.BattleReplayEntity, error)
	GetByGeometryIDs(ctx context.Context, geometryIDs []string) ([]*models.BattleReplayEntity, error)
	Create(ctx context.Context, params sqlc.CreateBattleReplayParams) (*models.BattleReplayEntity, error)
	Update(ctx context.Context, params sqlc.UpdateBattleReplayParams) (*models.BattleReplayEntity, error)
	Delete(ctx context.Context, id pgtype.UUID) error
	DeleteByIDs(ctx context.Context, ids []pgtype.UUID) error
	GetByProjectID(ctx context.Context, projectID pgtype.UUID) ([]*models.BattleReplayEntity, error)
	WithTx(tx pgx.Tx) BattleReplayRepository
}

type battleReplayRepository struct {
	q *sqlc.Queries
	c cache.Cache
}

func NewBattleReplayRepository(db sqlc.DBTX, c cache.Cache) BattleReplayRepository {
	return &battleReplayRepository{
		q: sqlc.New(db),
		c: c,
	}
}

func (r *battleReplayRepository) WithTx(tx pgx.Tx) BattleReplayRepository {
	return &battleReplayRepository{
		q: r.q.WithTx(tx),
		c: r.c,
	}
}

func (r *battleReplayRepository) rowToEntity(row sqlc.BattleReplay) *models.BattleReplayEntity {
	return &models.BattleReplayEntity{
		ID:                convert.UUIDToString(row.ID),
		GeometryID:        convert.UUIDToString(row.GeometryID),
		ProjectID:         convert.UUIDToString(row.ProjectID),
		TargetGeometryIDs: row.TargetGeometryIds,
		Detail:            row.Detail,
		IsDeleted:         row.IsDeleted,
		CreatedAt:         convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:         convert.TimeToPtr(row.UpdatedAt),
	}
}

func (r *battleReplayRepository) getByIDsWithFallback(ctx context.Context, ids []string) ([]*models.BattleReplayEntity, error) {
	if len(ids) == 0 {
		return []*models.BattleReplayEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("battle_replay:id:%s", id)
	}
	raws := r.c.MGet(ctx, keys...)

	var items []*models.BattleReplayEntity
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

	dbMap := make(map[string]*models.BattleReplayEntity)
	if len(missingPgIds) > 0 {
		dbRows, err := r.q.GetBattleReplaysByIDs(ctx, missingPgIds)
		if err == nil {
			for _, row := range dbRows {
				item := r.rowToEntity(row)
				dbMap[item.ID] = item
			}
		}
	}

	for i, b := range raws {
		if len(b) > 0 {
			var u models.BattleReplayEntity
			if err := json.Unmarshal(b, &u); err == nil {
				items = append(items, &u)
			}
		} else {
			if item, ok := dbMap[ids[i]]; ok {
				items = append(items, item)
				missingToCache[keys[i]] = item
			}
		}
	}

	if len(missingToCache) > 0 {
		_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
	}

	return items, nil
}

func (r *battleReplayRepository) GetByIDs(ctx context.Context, ids []string) ([]*models.BattleReplayEntity, error) {
	return r.getByIDsWithFallback(ctx, ids)
}

func (r *battleReplayRepository) GetByID(ctx context.Context, id pgtype.UUID) (*models.BattleReplayEntity, error) {
	cacheId := fmt.Sprintf("battle_replay:id:%s", convert.UUIDToString(id))
	var item models.BattleReplayEntity
	err := r.c.Get(ctx, cacheId, &item)
	if err == nil {
		_ = r.c.Set(ctx, cacheId, item, constants.NormalCacheDuration)
		return &item, nil
	}

	row, err := r.q.GetBattleReplayById(ctx, id)
	if err != nil {
		return nil, err
	}

	entity := r.rowToEntity(row)
	_ = r.c.Set(ctx, cacheId, entity, constants.NormalCacheDuration)

	return entity, nil
}

func (r *battleReplayRepository) GetByGeometryID(ctx context.Context, geometryID pgtype.UUID) ([]*models.BattleReplayEntity, error) {
	cacheKey := fmt.Sprintf("battle_replay:geometry:%s", convert.UUIDToString(geometryID))
	var cachedIDs []string
	err := r.c.Get(ctx, cacheKey, &cachedIDs)
	if err == nil {
		if len(cachedIDs) == 0 {
			return []*models.BattleReplayEntity{}, nil
		}
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.GetBattleReplaysByGeometryId(ctx, geometryID)
	if err != nil {
		return nil, err
	}

	var items []*models.BattleReplayEntity
	var ids []string
	itemToCache := make(map[string]any)

	for _, row := range rows {
		item := r.rowToEntity(row)
		ids = append(ids, item.ID)
		items = append(items, item)
		itemToCache[fmt.Sprintf("battle_replay:id:%s", item.ID)] = item
	}

	if len(itemToCache) > 0 {
		_ = r.c.MSet(ctx, itemToCache, constants.NormalCacheDuration)
	}
	_ = r.c.Set(ctx, cacheKey, ids, constants.ListCacheDuration)

	return items, nil
}

func (r *battleReplayRepository) GetByGeometryIDs(ctx context.Context, geometryIDs []string) ([]*models.BattleReplayEntity, error) {
	if len(geometryIDs) == 0 {
		return []*models.BattleReplayEntity{}, nil
	}

	var pgIds []pgtype.UUID
	for _, id := range geometryIDs {
		pgId := pgtype.UUID{}
		if err := pgId.Scan(id); err == nil {
			pgIds = append(pgIds, pgId)
		}
	}

	rows, err := r.q.GetBattleReplaysByGeometryIDs(ctx, pgIds)
	if err != nil {
		return nil, err
	}

	var items []*models.BattleReplayEntity
	itemToCache := make(map[string]any)

	for _, row := range rows {
		item := r.rowToEntity(row)
		items = append(items, item)
		itemToCache[fmt.Sprintf("battle_replay:id:%s", item.ID)] = item
	}

	if len(itemToCache) > 0 {
		_ = r.c.MSet(ctx, itemToCache, constants.NormalCacheDuration)
	}

	return items, nil
}

func (r *battleReplayRepository) GetByProjectID(ctx context.Context, projectID pgtype.UUID) ([]*models.BattleReplayEntity, error) {
	cacheKey := fmt.Sprintf("battle_replay:project:%s", convert.UUIDToString(projectID))
	var cachedIDs []string
	err := r.c.Get(ctx, cacheKey, &cachedIDs)
	if err == nil {
		if len(cachedIDs) == 0 {
			return []*models.BattleReplayEntity{}, nil
		}
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.GetBattleReplaysByProjectId(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var items []*models.BattleReplayEntity
	var ids []string
	itemToCache := make(map[string]any)

	for _, row := range rows {
		item := r.rowToEntity(row)
		ids = append(ids, item.ID)
		items = append(items, item)
		itemToCache[fmt.Sprintf("battle_replay:id:%s", item.ID)] = item
	}

	if len(itemToCache) > 0 {
		_ = r.c.MSet(ctx, itemToCache, constants.NormalCacheDuration)
	}
	_ = r.c.Set(ctx, cacheKey, ids, constants.ListCacheDuration)

	return items, nil
}

func (r *battleReplayRepository) Create(ctx context.Context, params sqlc.CreateBattleReplayParams) (*models.BattleReplayEntity, error) {
	row, err := r.q.CreateBattleReplay(ctx, params)
	if err != nil {
		return nil, err
	}

	entity := r.rowToEntity(row)

	_ = r.c.Del(ctx, fmt.Sprintf("battle_replay:project:%s", entity.ProjectID))
	_ = r.c.Del(ctx, fmt.Sprintf("battle_replay:geometry:%s", entity.GeometryID))

	return entity, nil
}

func (r *battleReplayRepository) Update(ctx context.Context, params sqlc.UpdateBattleReplayParams) (*models.BattleReplayEntity, error) {
	row, err := r.q.UpdateBattleReplay(ctx, params)
	if err != nil {
		return nil, err
	}

	entity := r.rowToEntity(row)
	_ = r.c.Del(ctx, fmt.Sprintf("battle_replay:id:%s", entity.ID))
	return entity, nil
}

func (r *battleReplayRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	err := r.q.DeleteBattleReplay(ctx, id)
	if err != nil {
		return err
	}
	_ = r.c.Del(ctx, fmt.Sprintf("battle_replay:id:%s", convert.UUIDToString(id)))
	return nil
}

func (r *battleReplayRepository) DeleteByIDs(ctx context.Context, ids []pgtype.UUID) error {
	err := r.q.DeleteBattleReplaysByIDs(ctx, ids)
	if err != nil {
		return err
	}
	if len(ids) > 0 {
		keys := make([]string, len(ids))
		for i, id := range ids {
			keys[i] = fmt.Sprintf("battle_replay:id:%s", convert.UUIDToString(id))
		}
		_ = r.c.Del(ctx, keys...)
	}
	return nil
}
