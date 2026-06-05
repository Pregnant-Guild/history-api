package repositories

import (
	"context"
	json "history-api/pkg/jsonx"

	"github.com/jackc/pgx/v5"
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
	AddMember(ctx context.Context, params sqlc.AddProjectMemberParams) error
	UpdateMemberRole(ctx context.Context, params sqlc.UpdateProjectMemberRoleParams) error
	RemoveMember(ctx context.Context, params sqlc.RemoveProjectMemberParams) error
	CheckPermission(ctx context.Context, params sqlc.CheckProjectPermissionParams) (int16, error)
	ChangeOwner(ctx context.Context, params sqlc.ChangeProjectOwnerParams) error
	UpdateLatestCommit(ctx context.Context, params sqlc.UpdateLatestCommitParams) error
	WithTx(tx pgx.Tx) ProjectRepository
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

func (r *projectRepository) WithTx(tx pgx.Tx) ProjectRepository {
	return &projectRepository{
		q: r.q.WithTx(tx),
		c: r.c,
	}
}
func (r *projectRepository) generateQueryKey(prefix string, params any) string {
	return cache.QueryKey(prefix, params)
}

func (r *projectRepository) getByIDsWithFallback(ctx context.Context, ids []string) ([]*models.ProjectEntity, error) {
	if len(ids) == 0 {
		return []*models.ProjectEntity{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = cache.Key("project:id", id)
	}
	raws := r.c.MGet(ctx, keys...)

	projects := make([]*models.ProjectEntity, 0, len(ids))
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

	dbMap := make(map[string]*models.ProjectEntity, len(missingPgIds))
	if len(missingPgIds) > 0 {
		dbRows, err := r.q.GetProjectsByIDs(ctx, missingPgIds)
		if err == nil {
			for _, row := range dbRows {
				item := models.ProjectEntity{
					ID:             convert.UUIDToString(row.ID),
					Title:          row.Title,
					Description:    convert.TextToString(row.Description),
					LatestCommitID: convert.UUIDToStringPtr(row.LatestCommitID),
					ProjectStatus:  constants.ParseProjectStatusType(row.ProjectStatus),
					LockedBy:       convert.UUIDToStringPtr(row.LockedBy),
					IsDeleted:      row.IsDeleted,
					UserID:         convert.UUIDToString(row.UserID),
					CreatedAt:      convert.TimeToPtr(row.CreatedAt),
					UpdatedAt:      convert.TimeToPtr(row.UpdatedAt),
				}
				_ = item.ParseUser(row.User)
				_ = item.ParseCommits(row.Commits)
				_ = item.ParseSubmissions(row.Submissions)
				_ = item.ParseMembers(row.Members)
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
	cacheId := cache.Key("project:id", convert.UUIDToString(id))
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
		ID:             convert.UUIDToString(row.ID),
		Title:          row.Title,
		Description:    convert.TextToString(row.Description),
		LatestCommitID: convert.UUIDToStringPtr(row.LatestCommitID),
		ProjectStatus:  constants.ParseProjectStatusType(row.ProjectStatus),
		LockedBy:       convert.UUIDToStringPtr(row.LockedBy),
		IsDeleted:      row.IsDeleted,
		UserID:         convert.UUIDToString(row.UserID),
		CreatedAt:      convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:      convert.TimeToPtr(row.UpdatedAt),
	}
	_ = project.ParseUser(row.User)
	_ = project.ParseCommits(row.Commits)
	_ = project.ParseSubmissions(row.Submissions)
	_ = project.ParseMembers(row.Members)

	_ = r.c.Set(ctx, cacheId, project, constants.NormalCacheDuration)

	return &project, nil
}

func (r *projectRepository) GetByUserID(ctx context.Context, params sqlc.GetProjectsByUserIdParams) ([]*models.ProjectEntity, error) {
	queryKey := r.generateQueryKey("project:user", params)
	var cachedIDs []string
	err := r.c.Get(ctx, queryKey, &cachedIDs)
	if err == nil {
		if len(cachedIDs) == 0 {
			return []*models.ProjectEntity{}, nil
		}
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.GetProjectsByUserId(ctx, params)
	if err != nil {
		return nil, err
	}

	projects := make([]*models.ProjectEntity, 0, len(rows))
	ids := make([]string, 0, len(rows))
	projectToCache := make(map[string]any, len(rows))

	for _, row := range rows {
		project := &models.ProjectEntity{
			ID:             convert.UUIDToString(row.ID),
			Title:          row.Title,
			Description:    convert.TextToString(row.Description),
			LatestCommitID: convert.UUIDToStringPtr(row.LatestCommitID),
			ProjectStatus:  constants.ParseProjectStatusType(row.ProjectStatus),
			LockedBy:       convert.UUIDToStringPtr(row.LockedBy),
			IsDeleted:      row.IsDeleted,
			UserID:         convert.UUIDToString(row.UserID),
			CreatedAt:      convert.TimeToPtr(row.CreatedAt),
			UpdatedAt:      convert.TimeToPtr(row.UpdatedAt),
		}
		_ = project.ParseUser(row.User)
		_ = project.ParseCommits(row.Commits)
		_ = project.ParseSubmissions(row.Submissions)
		_ = project.ParseMembers(row.Members)

		ids = append(ids, project.ID)
		projects = append(projects, project)
		projectToCache[cache.Key("project:id", project.ID)] = project
	}

	if len(projectToCache) > 0 {
		_ = r.c.MSet(ctx, projectToCache, constants.NormalCacheDuration)
	}
	_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)

	return projects, nil
}

func (r *projectRepository) Search(ctx context.Context, params sqlc.SearchProjectsParams) ([]*models.ProjectEntity, error) {
	queryKey := r.generateQueryKey("project:search", params)
	var cachedIDs []string
	err := r.c.Get(ctx, queryKey, &cachedIDs)
	if err == nil {
		if len(cachedIDs) == 0 {
			return []*models.ProjectEntity{}, nil
		}
		return r.getByIDsWithFallback(ctx, cachedIDs)
	}

	rows, err := r.q.SearchProjects(ctx, params)
	if err != nil {
		return nil, err
	}
	projects := make([]*models.ProjectEntity, 0, len(rows))
	ids := make([]string, 0, len(rows))
	projectToCache := make(map[string]any, len(rows))

	for _, row := range rows {
		project := &models.ProjectEntity{
			ID:             convert.UUIDToString(row.ID),
			Title:          row.Title,
			Description:    convert.TextToString(row.Description),
			LatestCommitID: convert.UUIDToStringPtr(row.LatestCommitID),
			ProjectStatus:  constants.ParseProjectStatusType(row.ProjectStatus),
			LockedBy:       convert.UUIDToStringPtr(row.LockedBy),
			IsDeleted:      row.IsDeleted,
			UserID:         convert.UUIDToString(row.UserID),
			CreatedAt:      convert.TimeToPtr(row.CreatedAt),
			UpdatedAt:      convert.TimeToPtr(row.UpdatedAt),
		}
		_ = project.ParseUser(row.User)
		_ = project.ParseCommits(row.Commits)
		_ = project.ParseSubmissions(row.Submissions)
		_ = project.ParseMembers(row.Members)

		ids = append(ids, project.ID)
		projects = append(projects, project)
		projectToCache[cache.Key("project:id", project.ID)] = project
	}

	if len(projectToCache) > 0 {
		_ = r.c.MSet(ctx, projectToCache, constants.NormalCacheDuration)
	}
	_ = r.c.Set(ctx, queryKey, ids, constants.ListCacheDuration)

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
		ID:             convert.UUIDToString(row.ID),
		Title:          row.Title,
		Description:    convert.TextToString(row.Description),
		LatestCommitID: convert.UUIDToStringPtr(row.LatestCommitID),
		ProjectStatus:  constants.ParseProjectStatusType(row.ProjectStatus),
		LockedBy:       convert.UUIDToStringPtr(row.LockedBy),
		IsDeleted:      row.IsDeleted,
		UserID:         convert.UUIDToString(row.UserID),
		CreatedAt:      convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:      convert.TimeToPtr(row.UpdatedAt),
	}
	_ = project.ParseUser(row.User)
	_ = project.ParseCommits(row.Commits)
	_ = project.ParseSubmissions(row.Submissions)
	_ = project.ParseMembers(row.Members)

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
		ID:             convert.UUIDToString(row.ID),
		Title:          row.Title,
		Description:    convert.TextToString(row.Description),
		LatestCommitID: convert.UUIDToStringPtr(row.LatestCommitID),
		ProjectStatus:  constants.ParseProjectStatusType(row.ProjectStatus),
		LockedBy:       convert.UUIDToStringPtr(row.LockedBy),
		IsDeleted:      row.IsDeleted,
		UserID:         convert.UUIDToString(row.UserID),
		CreatedAt:      convert.TimeToPtr(row.CreatedAt),
		UpdatedAt:      convert.TimeToPtr(row.UpdatedAt),
	}
	_ = project.ParseUser(row.User)
	_ = project.ParseCommits(row.Commits)
	_ = project.ParseSubmissions(row.Submissions)
	_ = project.ParseMembers(row.Members)

	_ = r.c.Del(ctx, cache.Key("project:id", project.ID))
	return &project, nil
}

func (r *projectRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	err := r.q.DeleteProject(ctx, id)
	if err != nil {
		return err
	}
	_ = r.c.Del(ctx, cache.Key("project:id", convert.UUIDToString(id)))
	go func() {
		bgCtx := context.Background()
		_ = r.c.DelByPattern(bgCtx, "project:search*")
		_ = r.c.DelByPattern(bgCtx, "project:user*")
		_ = r.c.DelByPattern(bgCtx, "project:count*")
	}()
	return nil
}

func (r *projectRepository) AddMember(ctx context.Context, params sqlc.AddProjectMemberParams) error {
	_, err := r.q.AddProjectMember(ctx, params)
	if err != nil {
		return err
	}
	go func() {
		_ = r.c.DelByPattern(context.Background(), "project:user*")
	}()
	_ = r.c.Del(ctx, cache.Key("project:id", convert.UUIDToString(params.ProjectID)))
	_ = r.c.Del(ctx, cache.Key2("project:perm", convert.UUIDToString(params.ProjectID), convert.UUIDToString(params.UserID)))
	return nil
}

func (r *projectRepository) UpdateMemberRole(ctx context.Context, params sqlc.UpdateProjectMemberRoleParams) error {
	_, err := r.q.UpdateProjectMemberRole(ctx, params)
	if err != nil {
		return err
	}
	go func() {
		_ = r.c.DelByPattern(context.Background(), "project:user*")
	}()
	_ = r.c.Del(ctx, cache.Key("project:id", convert.UUIDToString(params.ProjectID)))
	_ = r.c.Del(ctx, cache.Key2("project:perm", convert.UUIDToString(params.ProjectID), convert.UUIDToString(params.UserID)))
	return nil
}

func (r *projectRepository) RemoveMember(ctx context.Context, params sqlc.RemoveProjectMemberParams) error {
	err := r.q.RemoveProjectMember(ctx, params)
	if err != nil {
		return err
	}
	go func() {
		_ = r.c.DelByPattern(context.Background(), "project:user*")
	}()
	_ = r.c.Del(ctx, cache.Key("project:id", convert.UUIDToString(params.ProjectID)))
	_ = r.c.Del(ctx, cache.Key2("project:perm", convert.UUIDToString(params.ProjectID), convert.UUIDToString(params.UserID)))
	return nil
}

func (r *projectRepository) CheckPermission(ctx context.Context, params sqlc.CheckProjectPermissionParams) (int16, error) {
	cacheKey := cache.Key2("project:perm", convert.UUIDToString(params.ProjectID), convert.UUIDToString(params.UserID))
	var role int16
	if err := r.c.Get(ctx, cacheKey, &role); err == nil {
		return role, nil
	}

	role, err := r.q.CheckProjectPermission(ctx, params)
	if err != nil {
		return 0, err
	}

	_ = r.c.Set(ctx, cacheKey, role, constants.NormalCacheDuration)
	return role, nil
}

func (r *projectRepository) ChangeOwner(ctx context.Context, params sqlc.ChangeProjectOwnerParams) error {
	err := r.q.ChangeProjectOwner(ctx, params)
	if err != nil {
		return err
	}
	projectID := convert.UUIDToString(params.ID)
	_ = r.c.Del(ctx, cache.Key("project:id", projectID))
	go func() {
		_ = r.c.DelByPattern(context.Background(), "project:perm:"+projectID+":*")
	}()
	return nil
}

func (r *projectRepository) UpdateLatestCommit(ctx context.Context, params sqlc.UpdateLatestCommitParams) error {
	err := r.q.UpdateLatestCommit(ctx, params)
	if err != nil {
		return err
	}
	_ = r.c.Del(ctx, cache.Key("project:id", convert.UUIDToString(params.ID)))
	return nil
}
