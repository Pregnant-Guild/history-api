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

type WikiRepository interface {
	GetByID(ctx context.Context, id pgtype.UUID) (*models.WikiEntity, error)
	GetByIDs(ctx context.Context, ids []string) ([]*models.WikiEntity, error)
	GetBySlug(ctx context.Context, slug string) (*models.WikiEntity, error)
	GetBySlugs(ctx context.Context, slugs []string) ([]*models.WikiEntity, error)
	Search(ctx context.Context, params sqlc.SearchWikisParams) ([]*models.WikiEntity, error)
	Create(ctx context.Context, params sqlc.CreateWikiParams) (*models.WikiEntity, error)
	Update(ctx context.Context, params sqlc.UpdateWikiParams) (*models.WikiEntity, error)
	Delete(ctx context.Context, id pgtype.UUID) error
	CreateEntityWikis(ctx context.Context, params sqlc.CreateEntityWikisParams) error
	BulkDeleteEntityWikisByEntityId(ctx context.Context, entityId pgtype.UUID) error
	GetByProjectID(ctx context.Context, projectID pgtype.UUID) ([]*models.WikiEntity, error)
	DeleteByIDs(ctx context.Context, ids []pgtype.UUID) error
	BulkDeleteEntityWikisByWikiID(ctx context.Context, wikiID pgtype.UUID) error
	DeleteEntityWiki(ctx context.Context, entityID pgtype.UUID, wikiID pgtype.UUID) error
	DeleteEntityWikisByProjectID(ctx context.Context, projectID pgtype.UUID) error
	CreateContent(ctx context.Context, params sqlc.CreateWikiContentParams) (*models.WikiContentEntity, error)
	GetContentCountByWikiID(ctx context.Context, wikiID pgtype.UUID) (int64, error)
	GetContentByID(ctx context.Context, id pgtype.UUID) (*models.WikiContentEntity, error)
	GetContentByIDs(ctx context.Context, ids []string) ([]*models.WikiContentEntity, error)
	WithTx(tx pgx.Tx) WikiRepository
}

type wikiRepository struct {
	q *sqlc.Queries
	c cache.Cache
}

func NewWikiRepository(db sqlc.DBTX, c cache.Cache) WikiRepository {
	return &wikiRepository{
		q: sqlc.New(db),
		c: c,
	}
}

func (r *wikiRepository) WithTx(tx pgx.Tx) WikiRepository {
	return &wikiRepository{
		q: r.q.WithTx(tx),
		c: r.c,
	}
}

func (r *wikiRepository) generateQueryKey(prefix string, params any) string {
	b, _ := json.Marshal(params)
	hash := fmt.Sprintf("%x", md5.Sum(b))
	return fmt.Sprintf("%s:%s", prefix, hash)
}

func (r *wikiRepository) getByIDsWithFallback(ctx context.Context, ids []string) ([]*models.WikiEntity, error) {
	if len(ids) == 0 {
		return []*models.WikiEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("wiki:id:%s", id)
	}
	raws := r.c.MGet(ctx, keys...)

	var wikis []*models.WikiEntity
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

	dbMap := make(map[string]*models.WikiEntity)
	if len(missingPgIds) > 0 {
		dbRows, err := r.q.GetWikisByIDs(ctx, missingPgIds)
		if err == nil {
			for _, row := range dbRows {
				item := models.WikiEntity{
					ID:        convert.UUIDToString(row.ID),
					Title:     convert.TextToString(row.Title),
					Slug:      convert.TextToString(row.Slug),
					IsDeleted: row.IsDeleted,
					ProjectID: convert.UUIDToString(row.ProjectID),
					CreatedAt: convert.TimeToPtr(row.CreatedAt),
					UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
				}
				dbMap[item.ID] = &item
			}

			samples, err := r.q.GetWikiContentByWikiIDs(ctx, missingPgIds)
			if err == nil {
				for _, sample := range samples {
					wikiID := convert.UUIDToString(sample.WikiID)
					if item, ok := dbMap[wikiID]; ok {
						item.ContentSample = append(item.ContentSample, models.WikiContentSample{
							ID:        convert.UUIDToString(sample.ID),
							Title:     sample.Title,
							CreatedAt: convert.TimeToPtr(sample.CreatedAt),
						})
					}
				}
			}
		}
	}

	for i, b := range raws {
		if len(b) > 0 {
			var u models.WikiEntity
			if err := json.Unmarshal(b, &u); err == nil {
				wikis = append(wikis, &u)
			}
		} else {
			if item, ok := dbMap[ids[i]]; ok {
				wikis = append(wikis, item)
				missingToCache[keys[i]] = item
			}
		}
	}

	if len(missingToCache) > 0 {
		_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
	}

	return wikis, nil
}

func (r *wikiRepository) GetByIDs(ctx context.Context, ids []string) ([]*models.WikiEntity, error) {
	return r.getByIDsWithFallback(ctx, ids)
}

func (r *wikiRepository) GetByID(ctx context.Context, id pgtype.UUID) (*models.WikiEntity, error) {
	cacheId := fmt.Sprintf("wiki:id:%s", convert.UUIDToString(id))
	var wiki models.WikiEntity
	err := r.c.Get(ctx, cacheId, &wiki)
	if err == nil {
		_ = r.c.Set(ctx, cacheId, wiki, constants.NormalCacheDuration)
		return &wiki, nil
	}

	row, err := r.q.GetWikiById(ctx, id)
	if err != nil {
		return nil, err
	}

	wiki = models.WikiEntity{
		ID:        convert.UUIDToString(row.ID),
		Title:     convert.TextToString(row.Title),
		Slug:      convert.TextToString(row.Slug),
		IsDeleted: row.IsDeleted,
		ProjectID: convert.UUIDToString(row.ProjectID),
		CreatedAt: convert.TimeToPtr(row.CreatedAt),
		UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
	}

	samples, err := r.q.GetWikiContentByWikiID(ctx, row.ID)
	if err == nil {
		for _, sample := range samples {
			wiki.ContentSample = append(wiki.ContentSample, models.WikiContentSample{
				ID:        convert.UUIDToString(sample.ID),
				Title:     sample.Title,
				CreatedAt: convert.TimeToPtr(sample.CreatedAt),
			})
		}
	}

	_ = r.c.Set(ctx, cacheId, wiki, constants.NormalCacheDuration)

	return &wiki, nil
}

func (r *wikiRepository) Search(ctx context.Context, params sqlc.SearchWikisParams) ([]*models.WikiEntity, error) {
	queryKey := r.generateQueryKey("wiki:search", params)
	var cachedIDs []string
	if err := r.c.Get(ctx, queryKey, &cachedIDs); err == nil && len(cachedIDs) > 0 {
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.SearchWikis(ctx, params)
	if err != nil {
		return nil, err
	}
	var wikis []*models.WikiEntity
	var ids []string
	var pgIds []pgtype.UUID
	wikiMap := make(map[string]*models.WikiEntity)
	wikiToCache := make(map[string]any)

	for _, row := range rows {
		wiki := &models.WikiEntity{
			ID:        convert.UUIDToString(row.ID),
			Title:     convert.TextToString(row.Title),
			Slug:      convert.TextToString(row.Slug),
			IsDeleted: row.IsDeleted,
			ProjectID: convert.UUIDToString(row.ProjectID),
			CreatedAt: convert.TimeToPtr(row.CreatedAt),
			UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
		}
		ids = append(ids, wiki.ID)
		pgIds = append(pgIds, row.ID)
		wikis = append(wikis, wiki)
		wikiMap[wiki.ID] = wiki
	}

	if len(pgIds) > 0 {
		samples, err := r.q.GetWikiContentByWikiIDs(ctx, pgIds)
		if err == nil {
			for _, sample := range samples {
				wikiID := convert.UUIDToString(sample.WikiID)
				if item, ok := wikiMap[wikiID]; ok {
					item.ContentSample = append(item.ContentSample, models.WikiContentSample{
						ID:        convert.UUIDToString(sample.ID),
						Title:     sample.Title,
						CreatedAt: convert.TimeToPtr(sample.CreatedAt),
					})
				}
			}
		}
	}

	for _, wiki := range wikis {
		wikiToCache[fmt.Sprintf("wiki:id:%s", wiki.ID)] = wiki
	}
	if len(wikiToCache) > 0 {
		_ = r.c.MSet(ctx, wikiToCache, constants.NormalCacheDuration)
	}
	if len(ids) > 0 {
		_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)
	}

	return wikis, nil
}

func (r *wikiRepository) Create(ctx context.Context, params sqlc.CreateWikiParams) (*models.WikiEntity, error) {
	row, err := r.q.CreateWiki(ctx, params)
	if err != nil {
		return nil, err
	}

	wiki := models.WikiEntity{
		ID:        convert.UUIDToString(row.ID),
		Title:     convert.TextToString(row.Title),
		Slug:      convert.TextToString(row.Slug),
		IsDeleted: row.IsDeleted,
		ProjectID: convert.UUIDToString(row.ProjectID),
		CreatedAt: convert.TimeToPtr(row.CreatedAt),
		UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
	}

	return &wiki, nil
}

func (r *wikiRepository) Update(ctx context.Context, params sqlc.UpdateWikiParams) (*models.WikiEntity, error) {
	row, err := r.q.UpdateWiki(ctx, params)
	if err != nil {
		return nil, err
	}
	wiki := models.WikiEntity{
		ID:        convert.UUIDToString(row.ID),
		Title:     convert.TextToString(row.Title),
		Slug:      convert.TextToString(row.Slug),
		IsDeleted: row.IsDeleted,
		ProjectID: convert.UUIDToString(row.ProjectID),
		CreatedAt: convert.TimeToPtr(row.CreatedAt),
		UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
	}
	_ = r.c.Del(ctx, fmt.Sprintf("wiki:id:%s", wiki.ID))
	_ = r.c.Del(ctx, fmt.Sprintf("wiki:slug:%s", wiki.Slug))
	return &wiki, nil
}

func (r *wikiRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	err := r.q.DeleteWiki(ctx, id)
	if err != nil {
		return err
	}
	_ = r.c.Del(ctx, fmt.Sprintf("wiki:id:%s", convert.UUIDToString(id)))
	return nil
}

func (r *wikiRepository) CreateEntityWikis(ctx context.Context, params sqlc.CreateEntityWikisParams) error {
	return r.q.CreateEntityWikis(ctx, params)
}

func (r *wikiRepository) DeleteEntityWikisByProjectID(ctx context.Context, projectID pgtype.UUID) error {
	return r.q.DeleteEntityWikisByProjectID(ctx, projectID)
}

func (r *wikiRepository) BulkDeleteEntityWikisByEntityId(ctx context.Context, entityId pgtype.UUID) error {
	_, err := r.q.BulkDeleteEntityWikisByEntityId(ctx, entityId)
	if err != nil {
		return err
	}
	return nil
}

func (r *wikiRepository) GetByProjectID(ctx context.Context, projectID pgtype.UUID) ([]*models.WikiEntity, error) {
	cacheKey := fmt.Sprintf("wiki:project:%s", convert.UUIDToString(projectID))
	var cachedIDs []string
	if err := r.c.Get(ctx, cacheKey, &cachedIDs); err == nil && len(cachedIDs) > 0 {
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.GetWikisByProjectId(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var wikis []*models.WikiEntity
	var ids []string
	var pgIds []pgtype.UUID
	wikiMap := make(map[string]*models.WikiEntity)
	wikiToCache := make(map[string]any)

	for _, row := range rows {
		wiki := &models.WikiEntity{
			ID:        convert.UUIDToString(row.ID),
			Title:     convert.TextToString(row.Title),
			Slug:      convert.TextToString(row.Slug),
			IsDeleted: row.IsDeleted,
			ProjectID: convert.UUIDToString(row.ProjectID),
			CreatedAt: convert.TimeToPtr(row.CreatedAt),
			UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
		}
		ids = append(ids, wiki.ID)
		pgIds = append(pgIds, row.ID)
		wikis = append(wikis, wiki)
		wikiMap[wiki.ID] = wiki
	}

	if len(pgIds) > 0 {
		samples, err := r.q.GetWikiContentByWikiIDs(ctx, pgIds)
		if err == nil {
			for _, sample := range samples {
				wikiID := convert.UUIDToString(sample.WikiID)
				if item, ok := wikiMap[wikiID]; ok {
					item.ContentSample = append(item.ContentSample, models.WikiContentSample{
						ID:        convert.UUIDToString(sample.ID),
						Title:     sample.Title,
						CreatedAt: convert.TimeToPtr(sample.CreatedAt),
					})
				}
			}
		}
	}

	for _, wiki := range wikis {
		wikiToCache[fmt.Sprintf("wiki:id:%s", wiki.ID)] = wiki
	}
	if len(wikiToCache) > 0 {
		_ = r.c.MSet(ctx, wikiToCache, constants.NormalCacheDuration)
	}
	if len(ids) > 0 {
		_ = r.c.Set(ctx, cacheKey, ids, constants.ListCacheDuration)
	}

	return wikis, nil
}

func (r *wikiRepository) DeleteByIDs(ctx context.Context, ids []pgtype.UUID) error {
	err := r.q.DeleteWikisByIDs(ctx, ids)
	if err != nil {
		return err
	}
	if len(ids) > 0 {
		keys := make([]string, len(ids))
		for i, id := range ids {
			keys[i] = fmt.Sprintf("wiki:id:%s", convert.UUIDToString(id))
		}
		_ = r.c.Del(ctx, keys...)
	}
	return nil
}

func (r *wikiRepository) BulkDeleteEntityWikisByWikiID(ctx context.Context, wikiID pgtype.UUID) error {
	return r.q.BulkDeleteEntityWikisByWikiID(ctx, wikiID)
}

func (r *wikiRepository) DeleteEntityWiki(ctx context.Context, entityID pgtype.UUID, wikiID pgtype.UUID) error {
	return r.q.DeleteEntityWiki(ctx, sqlc.DeleteEntityWikiParams{
		EntityID: entityID,
		WikiID:   wikiID,
	})
}

func (r *wikiRepository) GetBySlug(ctx context.Context, slug string) (*models.WikiEntity, error) {
	cacheKey := fmt.Sprintf("wiki:slug:%s", slug)
	var wiki models.WikiEntity
	err := r.c.Get(ctx, cacheKey, &wiki)
	if err == nil {
		_ = r.c.Set(ctx, cacheKey, wiki, constants.NormalCacheDuration)
		return &wiki, nil
	}

	row, err := r.q.GetWikiBySlug(ctx, convert.StringToText(slug))
	if err != nil {
		return nil, err
	}

	wiki = models.WikiEntity{
		ID:        convert.UUIDToString(row.ID),
		Title:     convert.TextToString(row.Title),
		Slug:      convert.TextToString(row.Slug),
		IsDeleted: row.IsDeleted,
		ProjectID: convert.UUIDToString(row.ProjectID),
		CreatedAt: convert.TimeToPtr(row.CreatedAt),
		UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
	}

	samples, err := r.q.GetWikiContentByWikiID(ctx, row.ID)
	if err == nil {
		for _, sample := range samples {
			wiki.ContentSample = append(wiki.ContentSample, models.WikiContentSample{
				ID:        convert.UUIDToString(sample.ID),
				Title:     sample.Title,
				CreatedAt: convert.TimeToPtr(sample.CreatedAt),
			})
		}
	}

	_ = r.c.Set(ctx, cacheKey, wiki, constants.NormalCacheDuration)

	return &wiki, nil
}

func (r *wikiRepository) GetBySlugs(ctx context.Context, slugs []string) ([]*models.WikiEntity, error) {
	if len(slugs) == 0 {
		return []*models.WikiEntity{}, nil
	}
	keys := make([]string, len(slugs))
	for i, slug := range slugs {
		keys[i] = fmt.Sprintf("wiki:slug:%s", slug)
	}
	raws := r.c.MGet(ctx, keys...)

	var wikis []*models.WikiEntity
	missingToCache := make(map[string]any)
	var missingSlugs []string

	for i, b := range raws {
		if len(b) == 0 {
			missingSlugs = append(missingSlugs, slugs[i])
		}
	}

	dbMap := make(map[string]*models.WikiEntity)
	if len(missingSlugs) > 0 {
		dbRows, err := r.q.GetWikisBySlugs(ctx, missingSlugs)
		if err == nil {
			var pgIds []pgtype.UUID
			for _, row := range dbRows {
				item := models.WikiEntity{
					ID:        convert.UUIDToString(row.ID),
					Title:     convert.TextToString(row.Title),
					Slug:      convert.TextToString(row.Slug),
					IsDeleted: row.IsDeleted,
					ProjectID: convert.UUIDToString(row.ProjectID),
					CreatedAt: convert.TimeToPtr(row.CreatedAt),
					UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
				}
				pgIds = append(pgIds, row.ID)
				dbMap[item.Slug] = &item
			}

			if len(pgIds) > 0 {
				samples, sErr := r.q.GetWikiContentByWikiIDs(ctx, pgIds)
				if sErr == nil {
					wikiByID := make(map[string]*models.WikiEntity)
					for _, item := range dbMap {
						wikiByID[item.ID] = item
					}
					for _, sample := range samples {
						wikiID := convert.UUIDToString(sample.WikiID)
						if item, ok := wikiByID[wikiID]; ok {
							item.ContentSample = append(item.ContentSample, models.WikiContentSample{
								ID:        convert.UUIDToString(sample.ID),
								Title:     sample.Title,
								CreatedAt: convert.TimeToPtr(sample.CreatedAt),
							})
						}
					}
				}
			}
		}
	}

	for i, b := range raws {
		if len(b) > 0 {
			var u models.WikiEntity
			if err := json.Unmarshal(b, &u); err == nil {
				wikis = append(wikis, &u)
			}
		} else {
			if item, ok := dbMap[slugs[i]]; ok {
				wikis = append(wikis, item)
				missingToCache[keys[i]] = item
			}
		}
	}

	if len(missingToCache) > 0 {
		_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
	}

	return wikis, nil
}

func (r *wikiRepository) CreateContent(ctx context.Context, params sqlc.CreateWikiContentParams) (*models.WikiContentEntity, error) {
	row, err := r.q.CreateWikiContent(ctx, params)
	if err != nil {
		return nil, err
	}

	return &models.WikiContentEntity{
		ID:        convert.UUIDToString(row.ID),
		WikiID:    convert.UUIDToString(row.WikiID),
		Title:     row.Title,
		Content:   convert.TextToString(row.Content),
		Preview:   convert.TextToString(row.Preview),
		IsDeleted: row.IsDeleted,
		CreatedAt: convert.TimeToPtr(row.CreatedAt),
	}, nil
}

func (r *wikiRepository) GetContentCountByWikiID(ctx context.Context, wikiID pgtype.UUID) (int64, error) {
	return r.q.GetWikiContentCount(ctx, wikiID)
}

func (r *wikiRepository) getContentByIDsWithFallback(ctx context.Context, ids []string) ([]*models.WikiContentEntity, error) {
	if len(ids) == 0 {
		return []*models.WikiContentEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("wiki_content:id:%s", id)
	}
	raws := r.c.MGet(ctx, keys...)

	var contents []*models.WikiContentEntity
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

	dbMap := make(map[string]*models.WikiContentEntity)
	if len(missingPgIds) > 0 {
		dbRows, err := r.q.GetWikiContentByIDs(ctx, missingPgIds)
		if err == nil {
			for _, row := range dbRows {
				item := models.WikiContentEntity{
					ID:        convert.UUIDToString(row.ID),
					WikiID:    convert.UUIDToString(row.WikiID),
					Title:     row.Title,
					Content:   convert.TextToString(row.Content),
					Preview:   convert.TextToString(row.Preview),
					IsDeleted: row.IsDeleted,
					CreatedAt: convert.TimeToPtr(row.CreatedAt),
				}
				dbMap[item.ID] = &item
			}
		}
	}

	for i, b := range raws {
		if len(b) > 0 {
			var u models.WikiContentEntity
			if err := json.Unmarshal(b, &u); err == nil {
				contents = append(contents, &u)
			}
		} else {
			if item, ok := dbMap[ids[i]]; ok {
				contents = append(contents, item)
				missingToCache[keys[i]] = item
			}
		}
	}

	if len(missingToCache) > 0 {
		_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
	}

	return contents, nil
}

func (r *wikiRepository) GetContentByID(ctx context.Context, id pgtype.UUID) (*models.WikiContentEntity, error) {
	cacheId := fmt.Sprintf("wiki_content:id:%s", convert.UUIDToString(id))
	var content models.WikiContentEntity
	err := r.c.Get(ctx, cacheId, &content)
	if err == nil {
		_ = r.c.Set(ctx, cacheId, content, constants.NormalCacheDuration)
		return &content, nil
	}

	row, err := r.q.GetWikiContentById(ctx, id)
	if err != nil {
		return nil, err
	}

	content = models.WikiContentEntity{
		ID:        convert.UUIDToString(row.ID),
		WikiID:    convert.UUIDToString(row.WikiID),
		Title:     row.Title,
		Content:   convert.TextToString(row.Content),
		Preview:   convert.TextToString(row.Preview),
		IsDeleted: row.IsDeleted,
		CreatedAt: convert.TimeToPtr(row.CreatedAt),
	}
	_ = r.c.Set(ctx, cacheId, content, constants.NormalCacheDuration)

	return &content, nil
}

func (r *wikiRepository) GetContentByIDs(ctx context.Context, ids []string) ([]*models.WikiContentEntity, error) {
	return r.getContentByIDsWithFallback(ctx, ids)
}
