package repositories

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"

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
	Search(ctx context.Context, params sqlc.SearchWikisParams) ([]*models.WikiEntity, error)
	Create(ctx context.Context, params sqlc.CreateWikiParams) (*models.WikiEntity, error)
	Update(ctx context.Context, params sqlc.UpdateWikiParams) (*models.WikiEntity, error)
	Delete(ctx context.Context, id pgtype.UUID) error
	CreateEntityWikis(ctx context.Context, params sqlc.CreateEntityWikisParams) error
	BulkDeleteEntityWikisByEntityId(ctx context.Context, entityId pgtype.UUID) error
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

	for i, b := range raws {
		if len(b) > 0 {
			var w models.WikiEntity
			if err := json.Unmarshal(b, &w); err == nil {
				wikis = append(wikis, &w)
			}
		} else {
			pgId := pgtype.UUID{}
			err := pgId.Scan(ids[i])
			if err != nil {
				continue
			}
			dbWiki, err := r.GetByID(ctx, pgId)
			if err == nil && dbWiki != nil {
				wikis = append(wikis, dbWiki)
				missingToCache[keys[i]] = dbWiki
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
		Content:   convert.TextToString(row.Content),
		IsDeleted: row.IsDeleted,
		CreatedAt: convert.TimeToPtr(row.CreatedAt),
		UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
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
	wikiToCache := make(map[string]any)

	for _, row := range rows {
		wiki := &models.WikiEntity{
			ID:        convert.UUIDToString(row.ID),
			Title:     convert.TextToString(row.Title),
			Content:   convert.TextToString(row.Content),
			IsDeleted: row.IsDeleted,
			CreatedAt: convert.TimeToPtr(row.CreatedAt),
			UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
		}
		ids = append(ids, wiki.ID)
		wikis = append(wikis, wiki)
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
		Content:   convert.TextToString(row.Content),
		IsDeleted: row.IsDeleted,
		CreatedAt: convert.TimeToPtr(row.CreatedAt),
		UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
	}
	_ = r.c.Set(ctx, fmt.Sprintf("wiki:id:%s", wiki.ID), wiki, constants.NormalCacheDuration)

	go func() {
		bgCtx := context.Background()
		_ = r.c.DelByPattern(bgCtx, "wiki:search*")
	}()

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
		Content:   convert.TextToString(row.Content),
		IsDeleted: row.IsDeleted,
		CreatedAt: convert.TimeToPtr(row.CreatedAt),
		UpdatedAt: convert.TimeToPtr(row.UpdatedAt),
	}
	_ = r.c.Set(ctx, fmt.Sprintf("wiki:id:%s", wiki.ID), wiki, constants.NormalCacheDuration)
	return &wiki, nil
}

func (r *wikiRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	err := r.q.DeleteWiki(ctx, id)
	if err != nil {
		return err
	}
	_ = r.c.Del(ctx, fmt.Sprintf("wiki:id:%s", convert.UUIDToString(id)))
	go func() {
		_ = r.c.DelByPattern(context.Background(), "wiki:search*")
	}()
	return nil
}

func (r *wikiRepository) CreateEntityWikis(ctx context.Context, params sqlc.CreateEntityWikisParams) error {
	err := r.q.CreateEntityWikis(ctx, params)
	if err != nil {
		return err
	}
	return nil
}

func (r *wikiRepository) BulkDeleteEntityWikisByEntityId(ctx context.Context, entityId pgtype.UUID) error {
	wikiIDs, err := r.q.BulkDeleteEntityWikisByEntityId(ctx, entityId)
	if err != nil {
		return err
	}
	if len(wikiIDs) > 0 {
		keys := make([]string, len(wikiIDs))
		for i, id := range wikiIDs {
			keys[i] = fmt.Sprintf("wiki:id:%s", convert.UUIDToString(id))
		}
		go func() {
			_ = r.c.Del(context.Background(), keys...)
		}()
	}
	return nil
}
