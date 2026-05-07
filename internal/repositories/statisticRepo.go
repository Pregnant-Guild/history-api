package repositories

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/pkg/cache"
	"history-api/pkg/constants"
	"history-api/pkg/convert"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type StatisticRepository interface {
	Search(ctx context.Context, params sqlc.SearchSystemStatisticsParams) ([]*models.StatisticEntity, error)
	GetByDate(ctx context.Context, date time.Time) (*models.StatisticEntity, error)
	GetByID(ctx context.Context, id pgtype.UUID) (*models.StatisticEntity, error)
	Upsert(ctx context.Context, date time.Time) (*models.StatisticEntity, error)
	WithTx(tx pgx.Tx) StatisticRepository
}

type statisticRepository struct {
	q  *sqlc.Queries
	c  cache.Cache
	db sqlc.DBTX
}

func NewStatisticRepository(db sqlc.DBTX, c cache.Cache) StatisticRepository {
	return &statisticRepository{
		q:  sqlc.New(db),
		c:  c,
		db: db,
	}
}

func (r *statisticRepository) WithTx(tx pgx.Tx) StatisticRepository {
	return &statisticRepository{
		q:  r.q.WithTx(tx),
		c:  r.c,
		db: tx,
	}
}

func (r *statisticRepository) generateQueryKey(prefix string, params any) string {
	b, _ := json.Marshal(params)
	hash := fmt.Sprintf("%x", md5.Sum(b))
	return fmt.Sprintf("%s:%s", prefix, hash)
}

func mapToEntity(row sqlc.SystemStatistic) *models.StatisticEntity {
	return &models.StatisticEntity{
		ID:                convert.UUIDToString(row.ID),
		Date:              row.Date.Time,
		TotalUsers:        row.TotalUsers,
		TotalProjects:     row.TotalProjects,
		TotalCommits:      row.TotalCommits,
		TotalSubmissions:  row.TotalSubmissions,
		TotalMedias:       row.TotalMedias,
		TotalWikis:        row.TotalWikis,
		TotalEntities:     row.TotalEntities,
		TotalGeometries:   row.TotalGeometries,
		TotalStorageBytes: row.TotalStorageBytes,
		NewUsers:          row.NewUsers,
		NewProjects:       row.NewProjects,
		NewCommits:        row.NewCommits,
		NewSubmissions:    row.NewSubmissions,
		NewMedias:         row.NewMedias,
		NewWikis:          row.NewWikis,
		NewEntities:       row.NewEntities,
		NewGeometries:     row.NewGeometries,
		NewStorageBytes:   row.NewStorageBytes,
		CreatedAt:         convert.TimeToPtr(row.CreatedAt),
	}
}

func (r *statisticRepository) getByIDsWithFallback(ctx context.Context, ids []string) ([]*models.StatisticEntity, error) {
	if len(ids) == 0 {
		return []*models.StatisticEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("statistic:id:%s", id)
	}
	raws := r.c.MGet(ctx, keys...)

	var stats []*models.StatisticEntity
	missingStatsToCache := make(map[string]any)

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

	dbMap := make(map[string]*models.StatisticEntity)
	if len(missingPgIds) > 0 {
		dbRows, err := r.q.GetSystemStatisticsByIDs(ctx, missingPgIds)
		if err == nil {
			for _, row := range dbRows {
				entity := mapToEntity(row)
				dbMap[entity.ID] = entity
			}
		}
	}

	for i, b := range raws {
		if len(b) > 0 {
			var s models.StatisticEntity
			if err := json.Unmarshal(b, &s); err == nil {
				stats = append(stats, &s)
			}
		} else {
			if item, ok := dbMap[ids[i]]; ok {
				stats = append(stats, item)
				missingStatsToCache[keys[i]] = item
			}
		}
	}

	if len(missingStatsToCache) > 0 {
		_ = r.c.MSet(ctx, missingStatsToCache, constants.NormalCacheDuration)
	}

	return stats, nil
}

func (r *statisticRepository) Search(ctx context.Context, params sqlc.SearchSystemStatisticsParams) ([]*models.StatisticEntity, error) {
	queryKey := r.generateQueryKey("statistic:search", params)

	var cachedIDs []string
	if err := r.c.Get(ctx, queryKey, &cachedIDs); err == nil && len(cachedIDs) > 0 {
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.SearchSystemStatistics(ctx, params)
	if err != nil {
		return nil, err
	}

	var ids []string
	statsToCache := make(map[string]any)
	var stats []*models.StatisticEntity

	for _, row := range rows {
		entity := mapToEntity(row)
		ids = append(ids, entity.ID)
		stats = append(stats, entity)
		statsToCache[fmt.Sprintf("statistic:id:%s", entity.ID)] = entity
	}

	if len(statsToCache) > 0 {
		_ = r.c.MSet(ctx, statsToCache, constants.NormalCacheDuration)
	}
	if len(ids) > 0 {
		_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)
	}

	return stats, nil
}

func (r *statisticRepository) GetByID(ctx context.Context, id pgtype.UUID) (*models.StatisticEntity, error) {
	cacheId := fmt.Sprintf("statistic:id:%s", convert.UUIDToString(id))
	var stat models.StatisticEntity
	err := r.c.Get(ctx, cacheId, &stat)
	if err == nil {
		return &stat, nil
	}

	rows, err := r.q.GetSystemStatisticsByIDs(ctx, []pgtype.UUID{id})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	entity := mapToEntity(rows[0])
	_ = r.c.Set(ctx, cacheId, entity, constants.NormalCacheDuration)

	return entity, nil
}

func (r *statisticRepository) GetByDate(ctx context.Context, date time.Time) (*models.StatisticEntity, error) {
	dateStr := date.Format("2006-01-02")
	cacheId := fmt.Sprintf("statistic:date:%s", dateStr)

	var stat models.StatisticEntity
	err := r.c.Get(ctx, cacheId, &stat)
	if err == nil {
		return &stat, nil
	}

	row, err := r.q.GetSystemStatisticsByDate(ctx, pgtype.Date{Time: date, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	entity := mapToEntity(row)

	_ = r.c.Set(ctx, cacheId, entity, constants.NormalCacheDuration)
	_ = r.c.Set(ctx, fmt.Sprintf("statistic:id:%s", entity.ID), entity, constants.NormalCacheDuration)

	return entity, nil
}

func (r *statisticRepository) Upsert(ctx context.Context, date time.Time) (*models.StatisticEntity, error) {
	row, err := r.q.UpsertSystemStatistics(ctx, pgtype.Date{Time: date, Valid: true})
	if err != nil {
		return nil, err
	}

	entity := mapToEntity(row)

	go func() {
		bgCtx := context.Background()
		_ = r.c.DelByPattern(bgCtx, "statistic:search*")
		_ = r.c.Del(
			bgCtx,
			fmt.Sprintf("statistic:id:%s", entity.ID),
			fmt.Sprintf("statistic:date:%s", date.Format("2006-01-02")),
		)
	}()

	return entity, nil
}


