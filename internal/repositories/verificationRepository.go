package repositories

import (
	"context"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/pkg/cache"
	"history-api/pkg/constants"
	"history-api/pkg/convert"
	json "history-api/pkg/jsonx"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type VerificationRepository interface {
	GetByID(ctx context.Context, id pgtype.UUID) (*models.UserVerificationEntity, error)
	GetByUserID(ctx context.Context, id pgtype.UUID) ([]*models.UserVerificationEntity, error)
	Count(ctx context.Context, params sqlc.CountUserVerificationsParams) (int64, error)
	Create(ctx context.Context, params sqlc.CreateUserVerificationParams) (*models.UserVerificationEntity, error)
	UpdateStatus(ctx context.Context, params sqlc.UpdateUserVerificationStatusParams) error
	Search(ctx context.Context, params sqlc.SearchUserVerificationsParams) ([]*models.UserVerificationEntity, error)
	Delete(ctx context.Context, id pgtype.UUID) error
	CreateVerificationMedia(ctx context.Context, params sqlc.CreateVerificationMediaParams) error
	DeleteVerificationMedia(ctx context.Context, params sqlc.DeleteVerificationMediaParams) error
	BulkVerificationMediaByMediaId(ctx context.Context, mediaId pgtype.UUID) error
	WithTx(tx pgx.Tx) VerificationRepository
}

type verificationRepository struct {
	q *sqlc.Queries
	c cache.Cache
}

func NewVerificationRepository(db sqlc.DBTX, c cache.Cache) VerificationRepository {
	return &verificationRepository{
		q: sqlc.New(db),
		c: c,
	}
}

func (v *verificationRepository) WithTx(tx pgx.Tx) VerificationRepository {
	return &verificationRepository{
		q: v.q.WithTx(tx),
		c: v.c,
	}
}

func (v *verificationRepository) generateQueryKey(prefix string, params any) string {
	return cache.QueryKey(prefix, params)
}

func (v *verificationRepository) GetByID(ctx context.Context, id pgtype.UUID) (*models.UserVerificationEntity, error) {
	cacheId := cache.Key("verification:id", convert.UUIDToString(id))
	var verification models.UserVerificationEntity
	err := v.c.Get(ctx, cacheId, &verification)
	if err == nil {
		return &verification, nil
	}

	row, err := v.q.GetUserVerificationByID(ctx, id)
	if err != nil {
		return nil, err
	}

	verification = models.UserVerificationEntity{
		ID:         convert.UUIDToString(row.ID),
		VerifyType: constants.ParseVerifyType(row.VerifyType),
		Content:    convert.TextToString(row.Content),
		IsDeleted:  row.IsDeleted,
		Status:     constants.ParseStatusType(row.Status),
		ReviewNote: convert.TextToString(row.ReviewNote),
		ReviewedAt: convert.TimeToPtr(row.ReviewedAt),
		CreatedAt:  convert.TimeToPtr(row.CreatedAt),
	}

	if err := verification.ParseMedia(row.Medias); err != nil {
		return nil, err
	}

	if err := verification.ParseUser(row.User); err != nil {
		return nil, err
	}

	if err := verification.ParseReviewer(row.Reviewer); err != nil {
		return nil, err
	}

	_ = v.c.Set(ctx, cacheId, verification, constants.NormalCacheDuration)

	return &verification, nil
}

func (v *verificationRepository) getByIDsWithFallback(ctx context.Context, ids []string) ([]*models.UserVerificationEntity, error) {
	if len(ids) == 0 {
		return []*models.UserVerificationEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.Key("verification:id", id)
	}
	raws := v.c.MGet(ctx, keys...)

	verification := make([]*models.UserVerificationEntity, 0, len(ids))
	missingVerificationToCache := make(map[string]any, len(ids))

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

	dbMap := make(map[string]*models.UserVerificationEntity, len(missingPgIds))
	if len(missingPgIds) > 0 {
		dbRows, err := v.q.GetUserVerificationsByIDs(ctx, missingPgIds)
		if err == nil {
			for _, row := range dbRows {
				item := models.UserVerificationEntity{
					ID:         convert.UUIDToString(row.ID),
					VerifyType: constants.ParseVerifyType(row.VerifyType),
					Content:    convert.TextToString(row.Content),
					IsDeleted:  row.IsDeleted,
					Status:     constants.ParseStatusType(row.Status),
					ReviewNote: convert.TextToString(row.ReviewNote),
					ReviewedAt: convert.TimeToPtr(row.ReviewedAt),
					CreatedAt:  convert.TimeToPtr(row.CreatedAt),
				}
				_ = item.ParseMedia(row.Medias)
				_ = item.ParseUser(row.User)
				_ = item.ParseReviewer(row.Reviewer)
				dbMap[item.ID] = &item
			}
		}
	}

	for i, b := range raws {
		if len(b) > 0 {
			var u models.UserVerificationEntity
			if err := json.Unmarshal(b, &u); err == nil {
				verification = append(verification, &u)
			}
		} else {
			if item, ok := dbMap[ids[i]]; ok {
				verification = append(verification, item)
				missingVerificationToCache[keys[i]] = item
			}
		}
	}

	if len(missingVerificationToCache) > 0 {
		_ = v.c.MSet(ctx, missingVerificationToCache, constants.NormalCacheDuration)
	}

	return verification, nil
}

func (v *verificationRepository) Count(ctx context.Context, params sqlc.CountUserVerificationsParams) (int64, error) {
	queryKey := v.generateQueryKey("verification:count", params)
	var count int64
	if err := v.c.Get(ctx, queryKey, &count); err == nil {
		_ = v.c.Set(ctx, queryKey, count, constants.ListCacheDuration)
		return count, nil
	}
	count, err := v.q.CountUserVerifications(ctx, params)
	if err != nil {
		return 0, err
	}
	_ = v.c.Set(ctx, queryKey, count, constants.ListCacheDuration)
	return count, nil
}

func (v *verificationRepository) Create(ctx context.Context, params sqlc.CreateUserVerificationParams) (*models.UserVerificationEntity, error) {
	row, err := v.q.CreateUserVerification(ctx, params)
	if err != nil {
		return nil, err
	}
	verification := models.UserVerificationEntity{
		ID:         convert.UUIDToString(row.ID),
		VerifyType: constants.ParseVerifyType(row.VerifyType),
		Content:    convert.TextToString(row.Content),
		IsDeleted:  row.IsDeleted,
		Status:     constants.ParseStatusType(row.Status),
		ReviewNote: convert.TextToString(row.ReviewNote),
		ReviewedAt: convert.TimeToPtr(row.ReviewedAt),
		CreatedAt:  convert.TimeToPtr(row.CreatedAt),
	}

	if err := verification.ParseMedia(row.Medias); err != nil {
		return nil, err
	}

	if err := verification.ParseUser(row.User); err != nil {
		return nil, err
	}

	if err := verification.ParseReviewer(row.Reviewer); err != nil {
		return nil, err
	}

	go func() {
		bgCtx := context.Background()
		_ = v.c.DelByPattern(bgCtx, "verification:search*")
		_ = v.c.DelByPattern(bgCtx, "verification:count*")
	}()

	_ = v.c.Del(ctx, cache.Key("verification:userId", convert.UUIDToString(params.UserID)))

	return &verification, nil
}

func (v *verificationRepository) UpdateStatus(ctx context.Context, params sqlc.UpdateUserVerificationStatusParams) error {
	err := v.q.UpdateUserVerificationStatus(ctx, params)
	if err != nil {
		return err
	}
	_ = v.c.Del(ctx, cache.Key("verification:id", convert.UUIDToString(params.ID)))
	return nil
}

func (v *verificationRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	err := v.q.DeleteUserVerification(ctx, id)
	if err != nil {
		return err
	}

	_ = v.c.Del(ctx, cache.Key("verification:id", convert.UUIDToString(id)))
	go func() {
		_ = v.c.DelByPattern(context.Background(), "verification:count*")
	}()
	return nil
}

func (v *verificationRepository) BulkVerificationMediaByMediaId(ctx context.Context, mediaId pgtype.UUID) error {
	ids, err := v.q.BulkDeleteVerificationMediaByMediaId(ctx, mediaId)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return nil
	}

	listCacheId := make([]string, 0, len(ids))
	for _, it := range ids {
		id := convert.UUIDToString(it)
		if id == "" {
			continue
		}
		listCacheId = append(listCacheId, cache.Key("verification:id", id))
	}

	go func() {
		_ = v.c.Del(context.Background(), listCacheId...)
	}()

	return nil
}

func (v *verificationRepository) CreateVerificationMedia(ctx context.Context, params sqlc.CreateVerificationMediaParams) error {
	err := v.q.CreateVerificationMedia(ctx, params)
	return err
}

func (v *verificationRepository) DeleteVerificationMedia(ctx context.Context, params sqlc.DeleteVerificationMediaParams) error {
	err := v.q.DeleteVerificationMedia(ctx, params)
	return err
}

func (v *verificationRepository) GetByUserID(ctx context.Context, userId pgtype.UUID) ([]*models.UserVerificationEntity, error) {
	queryKey := cache.Key("verification:userId", convert.UUIDToString(userId))
	var cachedIDs []string
	err := v.c.Get(ctx, queryKey, &cachedIDs)
	if err == nil {
		if len(cachedIDs) == 0 {
			return []*models.UserVerificationEntity{}, nil
		}
		listItem, err := v.getByIDsWithFallback(ctx, cachedIDs)
		if err != nil {
			return nil, err
		}
		newCachedIDs := make([]string, len(listItem))
		for i, media := range listItem {
			newCachedIDs[i] = media.ID
		}
		_ = v.c.Set(ctx, queryKey, newCachedIDs, constants.ListCacheDuration)
		return listItem, nil
	}

	rows, err := v.q.GetUserVerifications(ctx, userId)
	if err != nil {
		return nil, err
	}
	items := make([]*models.UserVerificationEntity, 0, len(rows))
	ids := make([]string, 0, len(rows))
	itemToCache := make(map[string]any, len(rows))

	for _, row := range rows {
		verification := &models.UserVerificationEntity{
			ID:         convert.UUIDToString(row.ID),
			VerifyType: constants.ParseVerifyType(row.VerifyType),
			Content:    convert.TextToString(row.Content),
			IsDeleted:  row.IsDeleted,
			Status:     constants.ParseStatusType(row.Status),
			ReviewNote: convert.TextToString(row.ReviewNote),
			ReviewedAt: convert.TimeToPtr(row.ReviewedAt),
			CreatedAt:  convert.TimeToPtr(row.CreatedAt),
		}
		if err := verification.ParseMedia(row.Medias); err != nil {
			return nil, err
		}

		if err := verification.ParseUser(row.User); err != nil {
			return nil, err
		}

		if err := verification.ParseReviewer(row.Reviewer); err != nil {
			return nil, err
		}

		ids = append(ids, verification.ID)
		items = append(items, verification)

		itemToCache[cache.Key("verification:id", verification.ID)] = verification
	}

	if len(itemToCache) > 0 {
		_ = v.c.MSet(ctx, itemToCache, constants.NormalCacheDuration)
	}

	_ = v.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)

	return items, nil
}

func (v *verificationRepository) Search(ctx context.Context, params sqlc.SearchUserVerificationsParams) ([]*models.UserVerificationEntity, error) {
	queryKey := v.generateQueryKey("verification:search", params)
	var cachedIDs []string
	err := v.c.Get(ctx, queryKey, &cachedIDs)
	if err == nil {
		if len(cachedIDs) == 0 {
			return []*models.UserVerificationEntity{}, nil
		}
		listItem, err := v.getByIDsWithFallback(ctx, cachedIDs)
		if err != nil {
			return nil, err
		}
		newCachedIDs := make([]string, len(listItem))
		for i, media := range listItem {
			newCachedIDs[i] = media.ID
		}
		_ = v.c.Set(ctx, queryKey, newCachedIDs, constants.ListCacheDuration)
		return listItem, err
	}

	rows, err := v.q.SearchUserVerifications(ctx, params)
	if err != nil {
		return nil, err
	}
	items := make([]*models.UserVerificationEntity, 0, len(rows))
	ids := make([]string, 0, len(rows))
	itemToCache := make(map[string]any, len(rows))

	for _, row := range rows {
		verification := &models.UserVerificationEntity{
			ID:         convert.UUIDToString(row.ID),
			VerifyType: constants.ParseVerifyType(row.VerifyType),
			Content:    convert.TextToString(row.Content),
			IsDeleted:  row.IsDeleted,
			Status:     constants.ParseStatusType(row.Status),
			ReviewNote: convert.TextToString(row.ReviewNote),
			ReviewedAt: convert.TimeToPtr(row.ReviewedAt),
			CreatedAt:  convert.TimeToPtr(row.CreatedAt),
		}

		if err := verification.ParseMedia(row.Medias); err != nil {
			return nil, err
		}

		if err := verification.ParseUser(row.User); err != nil {
			return nil, err
		}

		if err := verification.ParseReviewer(row.Reviewer); err != nil {
			return nil, err
		}

		ids = append(ids, verification.ID)
		items = append(items, verification)

		itemToCache[cache.Key("verification:id", verification.ID)] = verification
	}

	if len(itemToCache) > 0 {
		_ = v.c.MSet(ctx, itemToCache, constants.NormalCacheDuration)
	}

	_ = v.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)

	return items, nil
}
