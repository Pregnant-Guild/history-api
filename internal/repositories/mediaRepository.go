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

type MediaRepository interface {
	GetByID(ctx context.Context, id pgtype.UUID) (*models.MediaEntity, error)
	GetByIDs(ctx context.Context, ids []string) ([]*models.MediaEntity, error)
	GetByUserID(ctx context.Context, userId pgtype.UUID) ([]*models.MediaEntity, error)
	Search(ctx context.Context, params sqlc.SearchMediasParams) ([]*models.MediaEntity, error)
	Count(ctx context.Context, params sqlc.CountMediasParams) (int64, error)
	Delete(ctx context.Context, id pgtype.UUID) error
	BulkDelete(ctx context.Context, ids []pgtype.UUID) error
	Create(ctx context.Context, params sqlc.CreateMediaParams) (*models.MediaEntity, error)
	WithTx(tx pgx.Tx) MediaRepository
}

type mediaRepository struct {
	q *sqlc.Queries
	c cache.Cache
}

func NewMediaRepository(db sqlc.DBTX, c cache.Cache) MediaRepository {
	return &mediaRepository{
		q: sqlc.New(db),
		c: c,
	}
}

func (r *mediaRepository) WithTx(tx pgx.Tx) MediaRepository {
	return &mediaRepository{
		q: r.q.WithTx(tx),
		c: r.c,
	}
}

func (r *mediaRepository) generateQueryKey(prefix string, params any) string {
	return cache.QueryKey(prefix, params)
}

func (r *mediaRepository) getByIDsWithFallback(ctx context.Context, ids []string) ([]*models.MediaEntity, error) {
	if len(ids) == 0 {
		return []*models.MediaEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.Key("media:id", id)
	}
	raws := r.c.MGet(ctx, keys...)

	medias := make([]*models.MediaEntity, 0, len(ids))
	missingMediasToCache := make(map[string]any, len(ids))

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

	dbMap := make(map[string]*models.MediaEntity, len(missingPgIds))
	if len(missingPgIds) > 0 {
		dbRows, err := r.q.GetMediaByIDs(ctx, missingPgIds)
		if err == nil {
			for _, row := range dbRows {
				item := models.MediaEntity{
					ID:           convert.UUIDToString(row.ID),
					UserID:       convert.UUIDToString(row.UserID),
					StorageKey:   row.StorageKey,
					OriginalName: row.OriginalName,
					MimeType:     row.MimeType,
					Size:         row.Size,
					FileMetadata: row.FileMetadata,
					CreatedAt:    convert.TimeToPtr(row.CreatedAt),
					UpdatedAt:    convert.TimeToPtr(row.UpdatedAt),
				}
				dbMap[item.ID] = &item
			}
		}
	}

	for i, b := range raws {
		if len(b) > 0 {
			var u models.MediaEntity
			if err := json.Unmarshal(b, &u); err == nil {
				medias = append(medias, &u)
			}
		} else {
			if item, ok := dbMap[ids[i]]; ok {
				medias = append(medias, item)
				missingMediasToCache[keys[i]] = item
			}
		}
	}

	if len(missingMediasToCache) > 0 {
		_ = r.c.MSet(ctx, missingMediasToCache, constants.NormalCacheDuration)
	}

	return medias, nil
}

func (r *mediaRepository) GetByIDs(ctx context.Context, ids []string) ([]*models.MediaEntity, error) {
	return r.getByIDsWithFallback(ctx, ids)
}

func (r *mediaRepository) GetByID(ctx context.Context, id pgtype.UUID) (*models.MediaEntity, error) {
	cacheId := cache.Key("media:id", convert.UUIDToString(id))
	var media models.MediaEntity
	err := r.c.Get(ctx, cacheId, &media)
	if err == nil {
		_ = r.c.Set(ctx, cacheId, media, constants.NormalCacheDuration)
		return &media, nil
	}

	row, err := r.q.GetMediaByID(ctx, id)
	if err != nil {
		return nil, err
	}

	media = models.MediaEntity{
		ID:           convert.UUIDToString(row.ID),
		UserID:       convert.UUIDToString(row.UserID),
		StorageKey:   row.StorageKey,
		OriginalName: row.OriginalName,
		MimeType:     row.MimeType,
		Size:         row.Size,
		FileMetadata: row.FileMetadata,
		CreatedAt:    convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:    convert.TimeToPtr(row.UpdatedAt),
	}

	_ = r.c.Set(ctx, cacheId, media, constants.NormalCacheDuration)

	return &media, nil
}

func (r *mediaRepository) Create(ctx context.Context, params sqlc.CreateMediaParams) (*models.MediaEntity, error) {
	row, err := r.q.CreateMedia(ctx, params)
	if err != nil {
		return nil, err
	}

	go func() {
		bgCtx := context.Background()
		_ = r.c.DelByPattern(bgCtx, "media:search*")
		_ = r.c.DelByPattern(bgCtx, "media:count*")
	}()

	_ = r.c.Del(ctx, cache.Key("media:userId", convert.UUIDToString(params.UserID)))

	media := models.MediaEntity{
		ID:           convert.UUIDToString(row.ID),
		UserID:       convert.UUIDToString(row.UserID),
		StorageKey:   row.StorageKey,
		OriginalName: row.OriginalName,
		MimeType:     row.MimeType,
		Size:         row.Size,
		FileMetadata: row.FileMetadata,
		CreatedAt:    convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:    convert.TimeToPtr(row.UpdatedAt),
	}

	return &media, nil
}

func (r *mediaRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	err := r.q.DeleteMedia(ctx, id)
	if err != nil {
		return err
	}

	cacheId := cache.Key("media:id", convert.UUIDToString(id))
	_ = r.c.Del(ctx, cacheId)
	go func() {
		_ = r.c.DelByPattern(context.Background(), "media:count*")
	}()
	return nil
}

func (r *mediaRepository) BulkDelete(ctx context.Context, ids []pgtype.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	err := r.q.DeleteMedias(ctx, ids)
	if err != nil {
		return err
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.Key("media:id", convert.UUIDToString(id))
	}
	_ = r.c.Del(ctx, keys...)

	return nil
}

func (r *mediaRepository) Search(ctx context.Context, params sqlc.SearchMediasParams) ([]*models.MediaEntity, error) {
	queryKey := r.generateQueryKey("media:search", params)
	var cachedIDs []string
	err := r.c.Get(ctx, queryKey, &cachedIDs)
	if err == nil {
		if len(cachedIDs) == 0 {
			return []*models.MediaEntity{}, nil
		}
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

	rows, err := r.q.SearchMedias(ctx, params)
	if err != nil {
		return nil, err
	}
	medias := make([]*models.MediaEntity, 0, len(rows))
	ids := make([]string, 0, len(rows))
	mediasToCache := make(map[string]any, len(rows))

	for _, row := range rows {
		media := &models.MediaEntity{
			ID:           convert.UUIDToString(row.ID),
			UserID:       convert.UUIDToString(row.UserID),
			StorageKey:   row.StorageKey,
			OriginalName: row.OriginalName,
			MimeType:     row.MimeType,
			Size:         row.Size,
			FileMetadata: row.FileMetadata,
			CreatedAt:    convert.TimeToPtr(row.CreatedAt),
			UpdatedAt:    convert.TimeToPtr(row.UpdatedAt),
		}
		ids = append(ids, media.ID)
		medias = append(medias, media)

		mediasToCache[cache.Key("media:id", media.ID)] = media
	}

	if len(mediasToCache) > 0 {
		_ = r.c.MSet(ctx, mediasToCache, constants.NormalCacheDuration)
	}

	_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)

	return medias, nil
}

func (r *mediaRepository) Count(ctx context.Context, params sqlc.CountMediasParams) (int64, error) {
	queryKey := r.generateQueryKey("media:count", params)
	var count int64
	if err := r.c.Get(ctx, queryKey, &count); err == nil {
		_ = r.c.Set(ctx, queryKey, count, constants.ListCacheDuration)
		return count, nil
	}
	count, err := r.q.CountMedias(ctx, params)
	if err != nil {
		return 0, err
	}
	_ = r.c.Set(ctx, queryKey, count, constants.ListCacheDuration)
	return count, nil
}

func (r *mediaRepository) GetByUserID(ctx context.Context, userId pgtype.UUID) ([]*models.MediaEntity, error) {
	queryKey := cache.Key("media:userId", convert.UUIDToString(userId))
	var cachedIDs []string
	err := r.c.Get(ctx, queryKey, &cachedIDs)
	if err == nil {
		if len(cachedIDs) == 0 {
			return []*models.MediaEntity{}, nil
		}
		listItem, err := r.getByIDsWithFallback(ctx, cachedIDs)
		if err != nil {
			return nil, err
		}
		newCachedIDs := make([]string, len(listItem))
		for i, media := range listItem {
			newCachedIDs[i] = media.ID
		}
		_ = r.c.Set(ctx, queryKey, newCachedIDs, constants.ListCacheDuration)
		return listItem, nil
	}

	rows, err := r.q.GetMediasByUserID(ctx, userId)
	if err != nil {
		return nil, err
	}
	medias := make([]*models.MediaEntity, 0, len(rows))
	ids := make([]string, 0, len(rows))
	mediasToCache := make(map[string]any, len(rows))

	for _, row := range rows {
		media := &models.MediaEntity{
			ID:           convert.UUIDToString(row.ID),
			UserID:       convert.UUIDToString(row.UserID),
			StorageKey:   row.StorageKey,
			OriginalName: row.OriginalName,
			MimeType:     row.MimeType,
			Size:         row.Size,
			FileMetadata: row.FileMetadata,
			CreatedAt:    convert.TimeToPtr(row.CreatedAt),
			UpdatedAt:    convert.TimeToPtr(row.UpdatedAt),
		}
		ids = append(ids, media.ID)
		medias = append(medias, media)

		mediasToCache[cache.Key("media:id", media.ID)] = media
	}

	if len(mediasToCache) > 0 {
		_ = r.c.MSet(ctx, mediasToCache, constants.NormalCacheDuration)
	}

	_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)

	return medias, nil
}
