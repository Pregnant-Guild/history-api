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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type CommitRepository interface {
	Create(ctx context.Context, params sqlc.CreateCommitParams) (*models.CommitEntity, error)
	GetByID(ctx context.Context, id pgtype.UUID) (*models.CommitEntity, error)
	GetByProjectID(ctx context.Context, projectID pgtype.UUID) ([]*models.CommitEntity, error)
	Search(ctx context.Context, params sqlc.SearchCommitsParams) ([]*models.CommitEntity, error)
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
	cacheId := fmt.Sprintf("commit:id:%s", convert.UUIDToString(id))
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
	b, _ := json.Marshal(params)
	hash := fmt.Sprintf("%x", md5.Sum(b))
	return fmt.Sprintf("%s:query:%s", prefix, hash)
}

func (r *commitRepository) getByIDsWithFallback(ctx context.Context, ids []string) ([]*models.CommitEntity, error) {
	if len(ids) == 0 {
		return []*models.CommitEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("commit:id:%s", id)
	}
	raws := r.c.MGet(ctx, keys...)

	var commits []*models.CommitEntity
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

	dbMap := make(map[string]*models.CommitEntity)
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
			if err := json.Unmarshal(b, &c); err == nil {
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
	queryKey := fmt.Sprintf("commit:project:%s", convert.UUIDToString(projectID))
	var cachedIDs []string
	if err := r.c.Get(ctx, queryKey, &cachedIDs); err == nil && len(cachedIDs) > 0 {
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.GetCommitsByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	entities := make([]*models.CommitEntity, 0, len(rows))
	for _, row := range rows {
		entities = append(entities, &models.CommitEntity{
			ID:           convert.UUIDToString(row.ID),
			ProjectID:    convert.UUIDToString(row.ProjectID),
			SnapshotJson: row.SnapshotJson,
			SnapshotHash: convert.TextToString(row.SnapshotHash),
			UserID:       convert.UUIDToString(row.UserID),
			EditSummary:  convert.TextToString(row.EditSummary),
			IsDeleted:    row.IsDeleted,
			CreatedAt:    convert.TimeToPtr(row.CreatedAt),
		})
	}
	return entities, nil
}

func (r *commitRepository) Search(ctx context.Context, params sqlc.SearchCommitsParams) ([]*models.CommitEntity, error) {
	queryKey := r.generateQueryKey("commit:search", params)
	var cachedIDs []string
	if err := r.c.Get(ctx, queryKey, &cachedIDs); err == nil && len(cachedIDs) > 0 {
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}
	rows, err := r.q.SearchCommits(ctx, params)
	if err != nil {
		return nil, err
	}
	var commits []*models.CommitEntity
	var ids []string
	commitToCache := make(map[string]any)

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
		commitToCache[fmt.Sprintf("commit:id:%s", commit.ID)] = commit
	}

	if len(commitToCache) > 0 {
		_ = r.c.MSet(ctx, commitToCache, constants.NormalCacheDuration)
	}
	if len(ids) > 0 {
		_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)
	}

	return commits, nil
}
