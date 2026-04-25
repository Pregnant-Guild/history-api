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

type ProjectRepository interface {
	GetByID(ctx context.Context, id pgtype.UUID) (*models.ProjectEntity, error)
	GetByIDs(ctx context.Context, ids []string) ([]*models.ProjectEntity, error)
	GetByUserID(ctx context.Context, params sqlc.GetProjectsByUserIdParams) ([]*models.ProjectEntity, error)
	Search(ctx context.Context, params sqlc.SearchProjectsParams) ([]*models.ProjectEntity, error)
	Count(ctx context.Context, params sqlc.CountProjectsParams) (int64, error)
	Create(ctx context.Context, params sqlc.CreateProjectParams) (*models.ProjectEntity, error)
	Update(ctx context.Context, params sqlc.UpdateProjectParams) (*models.ProjectEntity, error)
	Delete(ctx context.Context, id pgtype.UUID) error
}

type projectRepository struct {
	q *sqlc.Queries
	c cache.Cache
}

func NewProjectRepository(db sqlc.DBTX, c cache.Cache) ProjectRepository {
	return &projectRepository{
		q: sqlc.New(db),
		c: c,
	}
}

func (r *projectRepository) generateQueryKey(prefix string, params any) string {
	b, _ := json.Marshal(params)
	hash := fmt.Sprintf("%x", md5.Sum(b))
	return fmt.Sprintf("%s:%s", prefix, hash)
}


func (r *projectRepository) getByIDsWithFallback(ctx context.Context, ids []string) ([]*models.ProjectEntity, error) {
	if len(ids) == 0 {
		return []*models.ProjectEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprintf("project:id:%s", id)
	}
	raws := r.c.MGet(ctx, keys...)

	var projects []*models.ProjectEntity
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

	dbMap := make(map[string]*models.ProjectEntity)
	if len(missingPgIds) > 0 {
		dbRows, err := r.q.GetProjectsByIDs(ctx, missingPgIds)
		if err == nil {
			for _, row := range dbRows {
				item := models.ProjectEntity{
					ID:               convert.UUIDToString(row.ID),
					Title:            row.Title,
					Description:      convert.TextToString(row.Description),
					LatestRevisionID: convert.UUIDToStringPtr(row.LatestRevisionID),
					VersionCount:     row.VersionCount,
					ProjectStatus:    constants.ParseProjectStatusType(row.ProjectStatus),
					LockedBy:         convert.UUIDToStringPtr(row.LockedBy),
					IsDeleted:        row.IsDeleted,
					UserID:           convert.UUIDToString(row.UserID),
					CreatedAt:        convert.TimeToPtr(row.CreatedAt),
					UpdatedAt:        convert.TimeToPtr(row.UpdatedAt),
					CommitIds:        convert.ListUUIDToString(row.CommitIds),
					SubmissionIds:    convert.ListUUIDToString(row.SubmissionIds),
				}
				_ = item.ParseUser(row.User)
				dbMap[item.ID] = &item
			}
		}
	}

	for i, b := range raws {
		if len(b) > 0 {
			var p models.ProjectEntity
			if err := json.Unmarshal(b, &p); err == nil {
				projects = append(projects, &p)
			}
		} else {
			if item, ok := dbMap[ids[i]]; ok {
				projects = append(projects, item)
				missingToCache[keys[i]] = item
			}
		}
	}

	if len(missingToCache) > 0 {
		_ = r.c.MSet(ctx, missingToCache, constants.NormalCacheDuration)
	}

	return projects, nil
}

func (r *projectRepository) GetByIDs(ctx context.Context, ids []string) ([]*models.ProjectEntity, error) {
	return r.getByIDsWithFallback(ctx, ids)
}

func (r *projectRepository) GetByID(ctx context.Context, id pgtype.UUID) (*models.ProjectEntity, error) {
	cacheId := fmt.Sprintf("project:id:%s", convert.UUIDToString(id))
	var project models.ProjectEntity
	err := r.c.Get(ctx, cacheId, &project)
	if err == nil {
		_ = r.c.Set(ctx, cacheId, project, constants.NormalCacheDuration)
		return &project, nil
	}

	row, err := r.q.GetProjectById(ctx, id)
	if err != nil {
		return nil, err
	}

	project = models.ProjectEntity{
		ID:               convert.UUIDToString(row.ID),
		Title:            row.Title,
		Description:      convert.TextToString(row.Description),
		LatestRevisionID: convert.UUIDToStringPtr(row.LatestRevisionID),
		VersionCount:     row.VersionCount,
		ProjectStatus:    constants.ParseProjectStatusType(row.ProjectStatus),
		LockedBy:         convert.UUIDToStringPtr(row.LockedBy),
		IsDeleted:        row.IsDeleted,
		UserID:           convert.UUIDToString(row.UserID),
		CreatedAt:        convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:        convert.TimeToPtr(row.UpdatedAt),
		CommitIds:        convert.ListUUIDToString(row.CommitIds),
		SubmissionIds:    convert.ListUUIDToString(row.SubmissionIds),
	}
	_ = project.ParseUser(row.User)

	_ = r.c.Set(ctx, cacheId, project, constants.NormalCacheDuration)

	return &project, nil
}

func (r *projectRepository) GetByUserID(ctx context.Context, params sqlc.GetProjectsByUserIdParams) ([]*models.ProjectEntity, error) {
	queryKey := r.generateQueryKey("project:user", params)
	var cachedIDs []string
	if err := r.c.Get(ctx, queryKey, &cachedIDs); err == nil && len(cachedIDs) > 0 {
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.GetProjectsByUserId(ctx, params)
	if err != nil {
		return nil, err
	}
	
	var projects []*models.ProjectEntity
	var ids []string
	projectToCache := make(map[string]any)

	for _, row := range rows {
		project := &models.ProjectEntity{
			ID:               convert.UUIDToString(row.ID),
			Title:            row.Title,
			Description:      convert.TextToString(row.Description),
			LatestRevisionID: convert.UUIDToStringPtr(row.LatestRevisionID),
			VersionCount:     row.VersionCount,
			ProjectStatus:    constants.ParseProjectStatusType(row.ProjectStatus),
			LockedBy:         convert.UUIDToStringPtr(row.LockedBy),
			IsDeleted:        row.IsDeleted,
			UserID:           convert.UUIDToString(row.UserID),
			CreatedAt:        convert.TimeToPtr(row.CreatedAt),
			UpdatedAt:        convert.TimeToPtr(row.UpdatedAt),
			CommitIds:        convert.ListUUIDToString(row.CommitIds),
			SubmissionIds:    convert.ListUUIDToString(row.SubmissionIds),
		}
		_ = project.ParseUser(row.User)

		ids = append(ids, project.ID)
		projects = append(projects, project)
		projectToCache[fmt.Sprintf("project:id:%s", project.ID)] = project
	}

	if len(projectToCache) > 0 {
		_ = r.c.MSet(ctx, projectToCache, constants.NormalCacheDuration)
	}
	if len(ids) > 0 {
		_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)
	}

	return projects, nil
}

func (r *projectRepository) Search(ctx context.Context, params sqlc.SearchProjectsParams) ([]*models.ProjectEntity, error) {
	queryKey := r.generateQueryKey("project:search", params)
	var cachedIDs []string
	if err := r.c.Get(ctx, queryKey, &cachedIDs); err == nil && len(cachedIDs) > 0 {
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.SearchProjects(ctx, params)
	if err != nil {
		return nil, err
	}
	var projects []*models.ProjectEntity
	var ids []string
	projectToCache := make(map[string]any)

	for _, row := range rows {
		project := &models.ProjectEntity{
			ID:               convert.UUIDToString(row.ID),
			Title:            row.Title,
			Description:      convert.TextToString(row.Description),
			LatestRevisionID: convert.UUIDToStringPtr(row.LatestRevisionID),
			VersionCount:     row.VersionCount,
			ProjectStatus:    constants.ParseProjectStatusType(row.ProjectStatus),
			LockedBy:         convert.UUIDToStringPtr(row.LockedBy),
			IsDeleted:        row.IsDeleted,
			UserID:           convert.UUIDToString(row.UserID),
			CreatedAt:        convert.TimeToPtr(row.CreatedAt),
			UpdatedAt:        convert.TimeToPtr(row.UpdatedAt),
			CommitIds:        convert.ListUUIDToString(row.CommitIds),
			SubmissionIds:    convert.ListUUIDToString(row.SubmissionIds),
		}
		_ = project.ParseUser(row.User)

		ids = append(ids, project.ID)
		projects = append(projects, project)
		projectToCache[fmt.Sprintf("project:id:%s", project.ID)] = project
	}

	if len(projectToCache) > 0 {
		_ = r.c.MSet(ctx, projectToCache, constants.NormalCacheDuration)
	}
	if len(ids) > 0 {
		_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)
	}

	return projects, nil
}

func (r *projectRepository) Count(ctx context.Context, params sqlc.CountProjectsParams) (int64, error) {
	queryKey := r.generateQueryKey("project:count", params)
	var count int64
	if err := r.c.Get(ctx, queryKey, &count); err == nil {
		return count, nil
	}

	count, err := r.q.CountProjects(ctx, params)
	if err != nil {
		return 0, err
	}

	_ = r.c.Set(ctx, queryKey, count, constants.NormalCacheDuration)
	return count, nil
}

func (r *projectRepository) Create(ctx context.Context, params sqlc.CreateProjectParams) (*models.ProjectEntity, error) {
	row, err := r.q.CreateProject(ctx, params)
	if err != nil {
		return nil, err
	}

	project := models.ProjectEntity{
		ID:               convert.UUIDToString(row.ID),
		Title:            row.Title,
		Description:      convert.TextToString(row.Description),
		LatestRevisionID: convert.UUIDToStringPtr(row.LatestRevisionID),
		VersionCount:     row.VersionCount,
		ProjectStatus:    constants.ParseProjectStatusType(row.ProjectStatus),
		LockedBy:         convert.UUIDToStringPtr(row.LockedBy),
		IsDeleted:        row.IsDeleted,
		UserID:           convert.UUIDToString(row.UserID),
		CreatedAt:        convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:        convert.TimeToPtr(row.UpdatedAt),
		CommitIds:        convert.ListUUIDToString(row.CommitIds),
		SubmissionIds:    convert.ListUUIDToString(row.SubmissionIds),
	}
	_ = project.ParseUser(row.User)

	_ = r.c.Set(ctx, fmt.Sprintf("project:id:%s", project.ID), project, constants.NormalCacheDuration)

	go func() {
		bgCtx := context.Background()
		_ = r.c.DelByPattern(bgCtx, "project:search*")
		_ = r.c.DelByPattern(bgCtx, "project:user*")
		_ = r.c.DelByPattern(bgCtx, "project:count*")
	}()
	return &project, nil
}

func (r *projectRepository) Update(ctx context.Context, params sqlc.UpdateProjectParams) (*models.ProjectEntity, error) {
	row, err := r.q.UpdateProject(ctx, params)
	if err != nil {
		return nil, err
	}
	project := models.ProjectEntity{
		ID:               convert.UUIDToString(row.ID),
		Title:            row.Title,
		Description:      convert.TextToString(row.Description),
		LatestRevisionID: convert.UUIDToStringPtr(row.LatestRevisionID),
		VersionCount:     row.VersionCount,
		ProjectStatus:    constants.ParseProjectStatusType(row.ProjectStatus),
		LockedBy:         convert.UUIDToStringPtr(row.LockedBy),
		IsDeleted:        row.IsDeleted,
		UserID:           convert.UUIDToString(row.UserID),
		CreatedAt:        convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:        convert.TimeToPtr(row.UpdatedAt),
		CommitIds:        convert.ListUUIDToString(row.CommitIds),
		SubmissionIds:    convert.ListUUIDToString(row.SubmissionIds),
	}
	_ = project.ParseUser(row.User)

	_ = r.c.Set(ctx, fmt.Sprintf("project:id:%s", project.ID), project, constants.NormalCacheDuration)
	return &project, nil
}

func (r *projectRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	err := r.q.DeleteProject(ctx, id)
	if err != nil {
		return err
	}
	_ = r.c.Del(ctx, fmt.Sprintf("project:id:%s", convert.UUIDToString(id)))
	go func() {
		bgCtx := context.Background()
		_ = r.c.DelByPattern(bgCtx, "project:search*")
		_ = r.c.DelByPattern(bgCtx, "project:user*")
		_ = r.c.DelByPattern(bgCtx, "project:count*")
	}()
	return nil
}
