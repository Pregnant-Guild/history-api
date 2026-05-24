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

type SubmissionRepository interface {
	GetByID(ctx context.Context, id pgtype.UUID) (*models.SubmissionEntity, error)
	GetByIDs(ctx context.Context, ids []string) ([]*models.SubmissionEntity, error)
	Search(ctx context.Context, params sqlc.SearchSubmissionsParams) ([]*models.SubmissionEntity, error)
	Count(ctx context.Context, params sqlc.CountSubmissionsParams) (int64, error)
	Create(ctx context.Context, params sqlc.CreateSubmissionParams) (*models.SubmissionEntity, error)
	Update(ctx context.Context, params sqlc.UpdateSubmissionParams) (*models.SubmissionEntity, error)
	Delete(ctx context.Context, id pgtype.UUID) error
	GetLatestApprovedSubmissionExcluding(ctx context.Context, projectID pgtype.UUID, id pgtype.UUID) (*models.SubmissionEntity, error)
	WithTx(tx pgx.Tx) SubmissionRepository
}

type submissionRepository struct {
	q *sqlc.Queries
	c cache.Cache
}

func NewSubmissionRepository(db sqlc.DBTX, c cache.Cache) SubmissionRepository {
	return &submissionRepository{
		q: sqlc.New(db),
		c: c,
	}
}

func (r *submissionRepository) WithTx(tx pgx.Tx) SubmissionRepository {
	return &submissionRepository{
		q: r.q.WithTx(tx),
		c: r.c,
	}
}

func (r *submissionRepository) GetByID(ctx context.Context, id pgtype.UUID) (*models.SubmissionEntity, error) {
	cacheId := fmt.Sprintf("submission:id:%s", convert.UUIDToString(id))
	var submission models.SubmissionEntity
	err := r.c.Get(ctx, cacheId, &submission)
	if err == nil {
		return &submission, nil
	}

	row, err := r.q.GetSubmissionById(ctx, id)
	if err != nil {
		return nil, err
	}
	entity := &models.SubmissionEntity{
		ID:                 convert.UUIDToString(row.ID),
		ProjectID:          convert.UUIDToString(row.ProjectID),
		CommitID:           convert.UUIDToString(row.CommitID),
		UserID:             convert.UUIDToString(row.UserID),
		CreatedAt:          convert.TimeToPtr(row.CreatedAt),
		Status:             constants.ParseStatusType(row.Status),
		ReviewedBy:         convert.UUIDToStringPtr(row.ReviewedBy),
		ReviewedAt:         convert.TimeToPtr(row.ReviewedAt),
		ReviewNote:         convert.TextToPtr(row.ReviewNote),
		Content:            convert.TextToPtr(row.Content),
		IsDeleted:          row.IsDeleted,
		ProjectTitle:       row.ProjectTitle,
		ProjectDescription: convert.TextToPtr(row.ProjectDescription),
	}
	_ = entity.ParseUser(row.User)
	_ = entity.ParseReviewer(row.Reviewer)

	_ = r.c.Set(ctx, cacheId, entity, constants.NormalCacheDuration)
	return entity, nil
}

func (r *submissionRepository) generateQueryKey(prefix string, params any) string {
	b, _ := json.Marshal(params)
	hash := fmt.Sprintf("%x", md5.Sum(b))
	return fmt.Sprintf("%s:%s", prefix, hash)
}

func (r *submissionRepository) getByIDsWithFallback(ctx context.Context, ids []string) ([]*models.SubmissionEntity, error) {
	if len(ids) == 0 {
		return []*models.SubmissionEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("submission:id:%s", id)
	}
	raws := r.c.MGet(ctx, keys...)

	var submission []*models.SubmissionEntity
	missingSubmisstionToCache := make(map[string]any)

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

	dbMap := make(map[string]*models.SubmissionEntity)
	if len(missingPgIds) > 0 {
		dbRows, err := r.q.GetSubmissionsByIDs(ctx, missingPgIds)
		if err == nil {
			for _, row := range dbRows {
				entity := &models.SubmissionEntity{
					ID:                 convert.UUIDToString(row.ID),
					ProjectID:          convert.UUIDToString(row.ProjectID),
					CommitID:           convert.UUIDToString(row.CommitID),
					UserID:             convert.UUIDToString(row.UserID),
					CreatedAt:          convert.TimeToPtr(row.CreatedAt),
					Status:             constants.ParseStatusType(row.Status),
					ReviewedBy:         convert.UUIDToStringPtr(row.ReviewedBy),
					ReviewedAt:         convert.TimeToPtr(row.ReviewedAt),
					ReviewNote:         convert.TextToPtr(row.ReviewNote),
					Content:            convert.TextToPtr(row.Content),
					IsDeleted:          row.IsDeleted,
					ProjectTitle:       row.ProjectTitle,
					ProjectDescription: convert.TextToPtr(row.ProjectDescription),
				}
				_ = entity.ParseUser(row.User)
				_ = entity.ParseReviewer(row.Reviewer)
				dbMap[entity.ID] = entity
			}
		}
	}

	for i, b := range raws {
		if len(b) > 0 {
			var u models.SubmissionEntity
			if err := json.Unmarshal(b, &u); err == nil {
				submission = append(submission, &u)
			}
		} else {
			if item, ok := dbMap[ids[i]]; ok {
				submission = append(submission, item)
				missingSubmisstionToCache[keys[i]] = item
			}
		}
	}

	if len(missingSubmisstionToCache) > 0 {
		_ = r.c.MSet(ctx, missingSubmisstionToCache, constants.NormalCacheDuration)
	}

	return submission, nil
}

func (r *submissionRepository) GetByIDs(ctx context.Context, ids []string) ([]*models.SubmissionEntity, error) {
	return r.getByIDsWithFallback(ctx, ids)
}

func (r *submissionRepository) Search(ctx context.Context, params sqlc.SearchSubmissionsParams) ([]*models.SubmissionEntity, error) {
	queryKey := r.generateQueryKey("verification:search", params)
	var cachedIDs []string
	if err := r.c.Get(ctx, queryKey, &cachedIDs); err == nil && len(cachedIDs) > 0 {
		listItem, err := r.getByIDsWithFallback(ctx, cachedIDs)
		if err != nil {
			return nil, err
		}
		newCachedIDs := make([]string, len(listItem))
		for i, media := range listItem {
			newCachedIDs[i] = media.ID
		}
		_ = r.c.Set(ctx, queryKey, newCachedIDs, constants.ListCacheDuration)
		return listItem, err
	}

	rows, err := r.q.SearchSubmissions(ctx, params)
	if err != nil {
		return nil, err
	}
	var items []*models.SubmissionEntity
	var ids []string
	itemToCache := make(map[string]any)

	for _, row := range rows {
		entity := &models.SubmissionEntity{
			ID:                 convert.UUIDToString(row.ID),
			ProjectID:          convert.UUIDToString(row.ProjectID),
			CommitID:           convert.UUIDToString(row.CommitID),
			UserID:             convert.UUIDToString(row.UserID),
			CreatedAt:          convert.TimeToPtr(row.CreatedAt),
			Status:             constants.ParseStatusType(row.Status),
			ReviewedBy:         convert.UUIDToStringPtr(row.ReviewedBy),
			ReviewedAt:         convert.TimeToPtr(row.ReviewedAt),
			ReviewNote:         convert.TextToPtr(row.ReviewNote),
			Content:            convert.TextToPtr(row.Content),
			IsDeleted:          row.IsDeleted,
			ProjectTitle:       row.ProjectTitle,
			ProjectDescription: convert.TextToPtr(row.ProjectDescription),
		}
		_ = entity.ParseUser(row.User)
		_ = entity.ParseReviewer(row.Reviewer)

		ids = append(ids, entity.ID)
		items = append(items, entity)

		itemToCache[fmt.Sprintf("submission:id:%s", entity.ID)] = entity
	}

	if len(itemToCache) > 0 {
		_ = r.c.MSet(ctx, itemToCache, constants.NormalCacheDuration)
	}

	if len(ids) > 0 {
		_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)
	}

	return items, nil
}

func (r *submissionRepository) Count(ctx context.Context, params sqlc.CountSubmissionsParams) (int64, error) {
	queryKey := r.generateQueryKey("submission:count", params)
	var count int64
	if err := r.c.Get(ctx, queryKey, &count); err == nil {
		_ = r.c.Set(ctx, queryKey, count, constants.ListCacheDuration)
		return count, nil
	}
	count, err := r.q.CountSubmissions(ctx, params)
	if err != nil {
		return 0, err
	}
	_ = r.c.Set(ctx, queryKey, count, constants.ListCacheDuration)
	return count, nil
}

func (r *submissionRepository) Create(ctx context.Context, params sqlc.CreateSubmissionParams) (*models.SubmissionEntity, error) {
	row, err := r.q.CreateSubmission(ctx, params)
	if err != nil {
		return nil, err
	}
	submission := models.SubmissionEntity{
		ID:                 convert.UUIDToString(row.ID),
		ProjectID:          convert.UUIDToString(row.ProjectID),
		CommitID:           convert.UUIDToString(row.CommitID),
		UserID:             convert.UUIDToString(row.UserID),
		CreatedAt:          convert.TimeToPtr(row.CreatedAt),
		Status:             constants.ParseStatusType(row.Status),
		ReviewedBy:         convert.UUIDToStringPtr(row.ReviewedBy),
		ReviewedAt:         convert.TimeToPtr(row.ReviewedAt),
		ReviewNote:         convert.TextToPtr(row.ReviewNote),
		Content:            convert.TextToPtr(row.Content),
		IsDeleted:          row.IsDeleted,
		ProjectTitle:       row.ProjectTitle,
		ProjectDescription: convert.TextToPtr(row.ProjectDescription),
	}

	if err := submission.ParseUser(row.User); err != nil {
		return nil, err
	}

	if err := submission.ParseReviewer(row.Reviewer); err != nil {
		return nil, err
	}

	go func() {
		bgCtx := context.Background()
		_ = r.c.DelByPattern(bgCtx, "submission:search*")
		_ = r.c.DelByPattern(bgCtx, "submission:count*")
	}()

	return &submission, nil
}

func (r *submissionRepository) Update(ctx context.Context, params sqlc.UpdateSubmissionParams) (*models.SubmissionEntity, error) {
	row, err := r.q.UpdateSubmission(ctx, params)
	if err != nil {
		return nil, err
	}

	submission := models.SubmissionEntity{
		ID:                 convert.UUIDToString(row.ID),
		ProjectID:          convert.UUIDToString(row.ProjectID),
		CommitID:           convert.UUIDToString(row.CommitID),
		UserID:             convert.UUIDToString(row.UserID),
		CreatedAt:          convert.TimeToPtr(row.CreatedAt),
		Status:             constants.ParseStatusType(row.Status),
		ReviewedBy:         convert.UUIDToStringPtr(row.ReviewedBy),
		ReviewedAt:         convert.TimeToPtr(row.ReviewedAt),
		ReviewNote:         convert.TextToPtr(row.ReviewNote),
		Content:            convert.TextToPtr(row.Content),
		IsDeleted:          row.IsDeleted,
		ProjectTitle:       row.ProjectTitle,
		ProjectDescription: convert.TextToPtr(row.ProjectDescription),
	}

	if err := submission.ParseUser(row.User); err != nil {
		return nil, err
	}

	if err := submission.ParseReviewer(row.Reviewer); err != nil {
		return nil, err
	}

	_ = r.c.Del(ctx, fmt.Sprintf("submission:id:%s", submission.ID))

	return &submission, nil
}

func (r *submissionRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	err := r.q.DeleteSubmission(ctx, id)
	if err != nil {
		return err
	}

	_ = r.c.Del(ctx, fmt.Sprintf("submission:id:%s", convert.UUIDToString(id)))
	go func() {
		bgCtx := context.Background()
		_ = r.c.DelByPattern(bgCtx, "submission:search*")
		_ = r.c.DelByPattern(bgCtx, "submission:count*")
	}()
	return nil
}

func (r *submissionRepository) GetLatestApprovedSubmissionExcluding(ctx context.Context, projectID pgtype.UUID, id pgtype.UUID) (*models.SubmissionEntity, error) {
	row, err := r.q.GetLatestApprovedSubmissionExcluding(ctx, sqlc.GetLatestApprovedSubmissionExcludingParams{
		ProjectID: projectID,
		ID:        id,
	})
	if err != nil {
		return nil, err
	}

	entity := &models.SubmissionEntity{
		ID:         convert.UUIDToString(row.ID),
		ProjectID:  convert.UUIDToString(row.ProjectID),
		CommitID:   convert.UUIDToString(row.CommitID),
		UserID:     convert.UUIDToString(row.UserID),
		CreatedAt:  convert.TimeToPtr(row.CreatedAt),
		Status:     constants.ParseStatusType(row.Status),
		ReviewedBy: convert.UUIDToStringPtr(row.ReviewedBy),
		ReviewedAt: convert.TimeToPtr(row.ReviewedAt),
		ReviewNote: convert.TextToPtr(row.ReviewNote),
		Content:    convert.TextToPtr(row.Content),
		IsDeleted:  row.IsDeleted,
	}
	return entity, nil
}
