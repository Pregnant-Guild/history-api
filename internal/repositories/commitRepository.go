package repositories

import (
	"context"
	"encoding/json"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/pkg/cache"
	"history-api/pkg/constants"
	"history-api/pkg/convert"
	"history-api/pkg/jsonx"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type CommitRepository interface {
	Create(ctx context.Context, params sqlc.CreateCommitParams) (*models.CommitEntity, error)
	GetByID(ctx context.Context, id pgtype.UUID) (*models.CommitEntity, error)
	GetByProjectID(ctx context.Context, projectID pgtype.UUID) ([]*models.CommitEntity, error)
	Search(ctx context.Context, params sqlc.SearchCommitsParams) ([]*models.CommitEntity, error)
	UpdateSnapshot(ctx context.Context, id pgtype.UUID, snapshot json.RawMessage) (*models.CommitEntity, error)
	WithTx(tx pgx.Tx) CommitRepository
}

type commitRepository struct {
	q *sqlc.Queries
	c cache.Cache
}

func NewCommitRepository(db sqlc.DBTX, c cache.Cache) CommitRepository {
	return &commitRepository{
		q: sqlc.New(db),
		c: c,
	}
}

func (r *commitRepository) WithTx(tx pgx.Tx) CommitRepository {
	return &commitRepository{
		q: r.q.WithTx(tx),
		c: r.c,
	}
}

func (r *commitRepository) Create(ctx context.Context, params sqlc.CreateCommitParams) (*models.CommitEntity, error) {
	row, err := r.q.CreateCommit(ctx, params)
	if err != nil {
		return nil, err
	}

	return &models.CommitEntity{
		ID:           convert.UUIDToString(row.ID),
		ProjectID:    convert.UUIDToString(row.ProjectID),
		SnapshotJson: row.SnapshotJson,
		SnapshotHash: convert.TextToString(row.SnapshotHash),
		UserID:       convert.UUIDToString(row.UserID),
		EditSummary:  convert.TextToString(row.EditSummary),
		IsDeleted:    row.IsDeleted,
		CreatedAt:    convert.TimeToPtr(row.CreatedAt),
	}, nil
}

func (r *commitRepository) GetByID(ctx context.Context, id pgtype.UUID) (*models.CommitEntity, error) {
	cacheId := cache.Key("commit:id", convert.UUIDToString(id))
	var commit models.CommitEntity
	err := r.c.Get(ctx, cacheId, &commit)
	if err == nil {
		_ = r.c.Set(ctx, cacheId, commit, constants.NormalCacheDuration)
		return &commit, nil
	}

	row, err := r.q.GetCommitById(ctx, id)
	if err != nil {
		return nil, err
	}

	commit = models.CommitEntity{
		ID:           convert.UUIDToString(row.ID),
		ProjectID:    convert.UUIDToString(row.ProjectID),
		SnapshotJson: row.SnapshotJson,
		SnapshotHash: convert.TextToString(row.SnapshotHash),
		UserID:       convert.UUIDToString(row.UserID),
		EditSummary:  convert.TextToString(row.EditSummary),
		IsDeleted:    row.IsDeleted,
		CreatedAt:    convert.TimeToPtr(row.CreatedAt),
	}

	_ = r.c.Set(ctx, cacheId, commit, constants.NormalCacheDuration)

	return &commit, nil
}

func (r *commitRepository) generateQueryKey(prefix string, params any) string {
	return cache.QueryKey(prefix, params)
}

func (r *commitRepository) getByIDsWithFallback(ctx context.Context, ids []string) ([]*models.CommitEntity, error) {
	if len(ids) == 0 {
		return []*models.CommitEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.Key("commit:id", id)
	}
	raws := r.c.MGet(ctx, keys...)

	commits := make([]*models.CommitEntity, 0, len(ids))
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

	dbMap := make(map[string]*models.CommitEntity, len(missingPgIds))
	if len(missingPgIds) > 0 {
		dbRows, err := r.q.GetCommitsByIDs(ctx, missingPgIds)
		if err == nil {
			for _, row := range dbRows {
				item := models.CommitEntity{
					ID:           convert.UUIDToString(row.ID),
					ProjectID:    convert.UUIDToString(row.ProjectID),
					SnapshotJson: row.SnapshotJson,
					SnapshotHash: convert.TextToString(row.SnapshotHash),
					UserID:       convert.UUIDToString(row.UserID),
					EditSummary:  convert.TextToString(row.EditSummary),
					IsDeleted:    row.IsDeleted,
					CreatedAt:    convert.TimeToPtr(row.CreatedAt),
				}
				dbMap[item.ID] = &item
			}
		}
	}

	for i, b := range raws {
		if len(b) > 0 {
			var c models.CommitEntity
			if err := jsonx.Unmarshal(b, &c); err == nil {
				commits = append(commits, &c)
			}
		} else {
			if item, ok := dbMap[ids[i]]; ok {
				commits = append(commits, item)
				missingToCache[keys[i]] = item
			}
		}
	}

	if len(missingToCache) > 0 {
		_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
	}

	return commits, nil
}

func (r *commitRepository) GetByProjectID(ctx context.Context, projectID pgtype.UUID) ([]*models.CommitEntity, error) {
	queryKey := cache.Key("commit:project", convert.UUIDToString(projectID))
	var cachedIDs []string
	err := r.c.Get(ctx, queryKey, &cachedIDs)
	if err == nil {
		if len(cachedIDs) == 0 {
			return []*models.CommitEntity{}, nil
		}
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.GetCommitsByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	entities := make([]*models.CommitEntity, 0, len(rows))
	ids := make([]string, len(rows))
	commitToCache := make(map[string]any, len(rows))
	for i, row := range rows {
		item := &models.CommitEntity{
			ID:           convert.UUIDToString(row.ID),
			ProjectID:    convert.UUIDToString(row.ProjectID),
			SnapshotJson: row.SnapshotJson,
			SnapshotHash: convert.TextToString(row.SnapshotHash),
			UserID:       convert.UUIDToString(row.UserID),
			EditSummary:  convert.TextToString(row.EditSummary),
			IsDeleted:    row.IsDeleted,
			CreatedAt:    convert.TimeToPtr(row.CreatedAt),
		}
		entities = append(entities, item)
		ids[i] = item.ID
		commitToCache[cache.Key("commit:id", item.ID)] = item
	}
	if len(commitToCache) > 0 {
		_ = r.c.MSet(ctx, commitToCache, constants.NormalCacheDuration)
	}
	_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)
	return entities, nil
}

func (r *commitRepository) Search(ctx context.Context, params sqlc.SearchCommitsParams) ([]*models.CommitEntity, error) {
	queryKey := r.generateQueryKey("commit:search", params)
	var cachedIDs []string
	err := r.c.Get(ctx, queryKey, &cachedIDs)
	if err == nil {
		if len(cachedIDs) == 0 {
			return []*models.CommitEntity{}, nil
		}
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}
	rows, err := r.q.SearchCommits(ctx, params)
	if err != nil {
		return nil, err
	}
	commits := make([]*models.CommitEntity, 0, len(rows))
	ids := make([]string, 0, len(rows))
	commitToCache := make(map[string]any, len(rows))

	for _, row := range rows {
		commit := &models.CommitEntity{
			ID:           convert.UUIDToString(row.ID),
			ProjectID:    convert.UUIDToString(row.ProjectID),
			SnapshotJson: row.SnapshotJson,
			SnapshotHash: convert.TextToString(row.SnapshotHash),
			UserID:       convert.UUIDToString(row.UserID),
			EditSummary:  convert.TextToString(row.EditSummary),
			IsDeleted:    row.IsDeleted,
			CreatedAt:    convert.TimeToPtr(row.CreatedAt),
		}
		ids = append(ids, commit.ID)
		commits = append(commits, commit)
		commitToCache[cache.Key("commit:id", commit.ID)] = commit
	}

	if len(commitToCache) > 0 {
		_ = r.c.MSet(ctx, commitToCache, constants.NormalCacheDuration)
	}
	_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)

	return commits, nil
}

func (r *commitRepository) UpdateSnapshot(ctx context.Context, id pgtype.UUID, snapshot json.RawMessage) (*models.CommitEntity, error) {
	row, err := r.q.UpdateCommitSnapshot(ctx, sqlc.UpdateCommitSnapshotParams{
		ID:           id,
		SnapshotJson: snapshot,
	})
	if err != nil {
		return nil, err
	}
	r.c.Del(ctx, cache.Key("commit:id", convert.UUIDToString(id)))

	return &models.CommitEntity{
		ID:           convert.UUIDToString(row.ID),
		ProjectID:    convert.UUIDToString(row.ProjectID),
		SnapshotJson: row.SnapshotJson,
		SnapshotHash: convert.TextToString(row.SnapshotHash),
		UserID:       convert.UUIDToString(row.UserID),
		EditSummary:  convert.TextToString(row.EditSummary),
		IsDeleted:    row.IsDeleted,
		CreatedAt:    convert.TimeToPtr(row.CreatedAt),
	}, nil
}
