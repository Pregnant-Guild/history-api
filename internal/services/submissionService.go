package services

import (
	"context"
	"errors"
	"fmt"
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/internal/repositories"
	"history-api/pkg/ai"
	"history-api/pkg/cache"
	"history-api/pkg/constants"
	"history-api/pkg/convert"
	json "history-api/pkg/jsonx"
	"regexp"
	"slices"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

var blockquoteRegex = regexp.MustCompile(`(?is)<blockquote\b[^>]*>.*?</blockquote>`)

type SubmissionService interface {
	CreateSubmission(ctx context.Context, userID string, dto *request.CreateSubmissionDto) (*response.SubmissionResponse, *fiber.Error)
	UpdateSubmissionStatus(ctx context.Context, reviewerID string, submissionID string, dto *request.UpdateSubmissionStatusDto) (*response.SubmissionResponse, *fiber.Error)
	GetSubmissionByID(ctx context.Context, id string) (*response.SubmissionResponse, *fiber.Error)
	SearchSubmissions(ctx context.Context, dto *request.SearchSubmissionDto) (*response.PaginatedResponse, *fiber.Error)
	DeleteSubmission(ctx context.Context, userID string, id string, claims *response.JWTClaims) *fiber.Error
}

type submissionService struct {
	submissionRepo   repositories.SubmissionRepository
	projectRepo      repositories.ProjectRepository
	commitRepo       repositories.CommitRepository
	userRepo         repositories.UserRepository
	wikiRepo         repositories.WikiRepository
	geometryRepo     repositories.GeometryRepository
	entityRepo       repositories.EntityRepository
	battleReplayRepo repositories.BattleReplayRepository
	ragRepo          repositories.RagRepository
	ragUtils         *ai.RagUtils
	db               *pgxpool.Pool
	c                cache.Cache
}

func NewSubmissionService(
	submissionRepo repositories.SubmissionRepository,
	projectRepo repositories.ProjectRepository,
	commitRepo repositories.CommitRepository,
	userRepo repositories.UserRepository,
	wikiRepo repositories.WikiRepository,
	geometryRepo repositories.GeometryRepository,
	entityRepo repositories.EntityRepository,
	battleReplayRepo repositories.BattleReplayRepository,
	ragRepo repositories.RagRepository,
	ragUtils *ai.RagUtils,
	db *pgxpool.Pool,
	c cache.Cache,
) SubmissionService {
	return &submissionService{
		submissionRepo:   submissionRepo,
		projectRepo:      projectRepo,
		commitRepo:       commitRepo,
		userRepo:         userRepo,
		wikiRepo:         wikiRepo,
		geometryRepo:     geometryRepo,
		entityRepo:       entityRepo,
		battleReplayRepo: battleReplayRepo,
		ragRepo:          ragRepo,
		ragUtils:         ragUtils,
		db:               db,
		c:                c,
	}
}

func (s *submissionService) CreateSubmission(ctx context.Context, userID string, dto *request.CreateSubmissionDto) (*response.SubmissionResponse, *fiber.Error) {
	projectUUID, err := convert.StringToUUID(dto.ProjectID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid project ID")
	}

	commitUUID, err := convert.StringToUUID(dto.CommitID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid commit ID")
	}

	userUUID, err := convert.StringToUUID(userID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid user ID")
	}

	commit, err := s.commitRepo.GetByID(ctx, commitUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Commit not found")
	}
	if commit.ProjectID != dto.ProjectID {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Commit does not belong to project")
	}

	var snapshotData request.CommitSnapshot
	if err := json.Unmarshal(commit.SnapshotJson, &snapshotData); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to parse commit snapshot")
	}

	entitySlugs := make([]string, 0, len(snapshotData.Entities))
	entitySlugToID := make(map[string]string, len(snapshotData.Entities))
	for _, entity := range snapshotData.Entities {
		if entity.Source == "inline" && entity.Slug != nil {
			entitySlugs = append(entitySlugs, *entity.Slug)
			entitySlugToID[*entity.Slug] = entity.ID
		}
	}

	if len(entitySlugs) > 0 {
		existEntities, err := s.entityRepo.GetBySlugs(ctx, entitySlugs)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to get entities")
		}
		for _, exist := range existEntities {
			if snapID, ok := entitySlugToID[exist.Slug]; ok {
				if exist.ID != snapID {
					return nil, fiber.NewError(fiber.StatusConflict, fmt.Sprintf("Entity %s already exists", exist.Slug))
				}
			}
		}
	}

	wikiSlugs := make([]string, 0, len(snapshotData.Wikis))
	wikiSlugToID := make(map[string]string, len(snapshotData.Wikis))
	for _, wiki := range snapshotData.Wikis {
		if wiki.Source == "inline" && wiki.Slug != nil {
			wikiSlugs = append(wikiSlugs, *wiki.Slug)
			wikiSlugToID[*wiki.Slug] = wiki.ID
		}
	}

	if len(wikiSlugs) > 0 {
		existWikis, err := s.wikiRepo.GetBySlugs(ctx, wikiSlugs)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to get wikis")
		}
		for _, exist := range existWikis {
			if snapID, ok := wikiSlugToID[exist.Slug]; ok {
				if exist.ID != snapID {
					return nil, fiber.NewError(fiber.StatusConflict, fmt.Sprintf("Wiki %s already exists", exist.Slug))
				}
			}
		}
	}

	project, err := s.projectRepo.GetByID(ctx, projectUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Project not found")
	}

	for _, sub := range project.Submissions {
		if sub.Status == constants.StatusTypePending {
			return nil, fiber.NewError(fiber.StatusConflict, "There is already a pending submission for this project")
		}
	}

	arg := sqlc.CreateSubmissionParams{
		ProjectID: projectUUID,
		CommitID:  commitUUID,
		UserID:    userUUID,
		Status:    constants.StatusTypePending.Int16(),
		Content:   convert.StringToText(dto.Content),
	}

	submission, err := s.submissionRepo.Create(ctx, arg)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create submission")
	}

	_ = s.c.Del(ctx, cache.Key("project:id", project.ID))

	return submission.ToResponse(), nil
}

func (s *submissionService) UpdateSubmissionStatus(ctx context.Context, reviewerID string, submissionID string, dto *request.UpdateSubmissionStatusDto) (*response.SubmissionResponse, *fiber.Error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer tx.Rollback(ctx)

	submissionRepo := s.submissionRepo.WithTx(tx)

	submissionUUID, err := convert.StringToUUID(submissionID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid submission ID")
	}

	reviewerUUID, err := convert.StringToUUID(reviewerID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid reviewer ID")
	}

	status := constants.ParseStatusTypeText(dto.Status)
	if status == constants.StatusTypeUnknown {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid status")
	}

	submission, err := s.submissionRepo.GetByID(ctx, submissionUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Submission not found")
	}

	if submission.Status != constants.StatusTypePending {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Submission already processed")
	}

	commitUUID, err := convert.StringToUUID(submission.CommitID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid commit ID")
	}

	commit, err := s.commitRepo.GetByID(ctx, commitUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Commit not found")
	}

	if commit.ProjectID != submission.ProjectID {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Commit does not belong to project")
	}

	var snapshotData request.CommitSnapshot
	err = json.Unmarshal(commit.SnapshotJson, &snapshotData)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to parse commit snapshot")
	}

	if status == constants.StatusTypeApproved {
		projectUUID, err := convert.StringToUUID(commit.ProjectID)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid project ID")
		}

		if err := s.applySnapshot(ctx, tx, projectUUID, commitUUID, &snapshotData); err != nil {
			return nil, err
		}
	}

	arg := sqlc.UpdateSubmissionParams{
		ID:         submissionUUID,
		Status:     pgtype.Int2{Int16: status.Int16(), Valid: true},
		ReviewedBy: reviewerUUID,
		ReviewNote: convert.StringToText(dto.ReviewNote),
	}

	updatedSubmission, err := submissionRepo.Update(ctx, arg)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to update submission status: "+err.Error())
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to commit transaction: "+err.Error())
	}

	if status == constants.StatusTypeApproved {
		go func() {
			bgCtx := context.Background()
			_ = s.c.DelByPattern(bgCtx, "entity:search*")
			_ = s.c.DelByPattern(bgCtx, "geometry:search*")
			_ = s.c.DelByPattern(bgCtx, "geometry:search:entity*")
			_ = s.c.DelByPattern(bgCtx, "wiki:search*")
		}()
	}

	_ = s.c.Del(ctx,
		cache.Key("project:id", submission.ProjectID),
		cache.Key("entity:project", submission.ProjectID),
		cache.Key("geometry:project", submission.ProjectID),
		cache.Key("wiki:project", submission.ProjectID),
		cache.Key("battle_replay:project", submission.ProjectID),
	)

	return updatedSubmission.ToResponse(), nil
}

func (m *submissionService) fillSearchArgs(arg *sqlc.SearchSubmissionsParams, dto *request.SearchSubmissionDto) {
	if dto.Sort != "" {
		arg.Sort = pgtype.Text{String: dto.Sort, Valid: true}
	} else {
		arg.Sort = pgtype.Text{String: "id", Valid: true}
	}

	arg.Order = pgtype.Text{String: "asc", Valid: true}
	if dto.Order == "desc" {
		arg.Order = pgtype.Text{String: "desc", Valid: true}
	}

	if len(dto.Statuses) > 0 {
		arg.Statuses = make([]int16, 0, len(dto.Statuses))
		for _, id := range dto.Statuses {
			if u := constants.ParseStatusTypeText(id); u != constants.StatusTypeUnknown {
				arg.Statuses = append(arg.Statuses, u.Int16())
			}
		}
	}

	if len(dto.UserIDs) > 0 {
		arg.UserIds = make([]pgtype.UUID, 0, len(dto.UserIDs))
		for _, id := range dto.UserIDs {
			if u, err := convert.StringToUUID(id); err == nil {
				arg.UserIds = append(arg.UserIds, u)
			}
		}
	}

	if dto.ReviewedBy != nil {
		if rvID, err := convert.StringToUUID(*dto.ReviewedBy); err == nil {
			arg.ReviewedBy = rvID
		}
	}

	if dto.CreatedFrom != nil {
		arg.CreatedFrom = pgtype.Timestamptz{Time: *dto.CreatedFrom, Valid: true}
	}

	if dto.CreatedTo != nil {
		arg.CreatedTo = pgtype.Timestamptz{Time: *dto.CreatedTo, Valid: true}
	}

	if dto.Search != "" {
		arg.SearchText = pgtype.Text{String: dto.Search, Valid: true}
	}
}

func (s *submissionService) GetSubmissionByID(ctx context.Context, id string) (*response.SubmissionResponse, *fiber.Error) {
	submissionUUID, err := convert.StringToUUID(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid submission ID")
	}

	submission, err := s.submissionRepo.GetByID(ctx, submissionUUID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Submission not found")
	}

	return submission.ToResponse(), nil
}

func (s *submissionService) SearchSubmissions(ctx context.Context, dto *request.SearchSubmissionDto) (*response.PaginatedResponse, *fiber.Error) {
	if dto.Page < 1 {
		dto.Page = 1
	}
	if dto.Limit == 0 {
		dto.Limit = 20
	}
	offset := (dto.Page - 1) * dto.Limit

	arg := sqlc.SearchSubmissionsParams{
		Limit:  int32(dto.Limit),
		Offset: int32(offset),
	}

	s.fillSearchArgs(&arg, dto)

	var rows []*models.SubmissionEntity
	var totalRecords int64

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		rows, err = s.submissionRepo.Search(gCtx, arg)
		return err
	})

	g.Go(func() error {
		countArg := sqlc.CountSubmissionsParams{
			UserIds:     arg.UserIds,
			Statuses:    arg.Statuses,
			ReviewedBy:  arg.ReviewedBy,
			CreatedFrom: arg.CreatedFrom,
			CreatedTo:   arg.CreatedTo,
			SearchText:  arg.SearchText,
		}
		var err error
		totalRecords, err = s.submissionRepo.Count(gCtx, countArg)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to search submissions")
	}

	submissions := models.SubmissionsEntityToResponse(rows)

	return response.BuildPaginatedResponse(submissions, totalRecords, dto.Page, dto.Limit), nil
}

func (s *submissionService) DeleteSubmission(ctx context.Context, userID string, id string, claims *response.JWTClaims) *fiber.Error {
	submissionUUID, err := convert.StringToUUID(id)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid submission ID")
	}

	submission, err := s.submissionRepo.GetByID(ctx, submissionUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Submission not found")
	}

	shoudDelete := false
	if slices.Contains(claims.Roles, constants.RoleTypeAdmin) || slices.Contains(claims.Roles, constants.RoleTypeMod) {
		shoudDelete = true
	}

	if submission.UserID == claims.UId && submission.Status == constants.StatusTypePending {
		shoudDelete = true
	}

	if !shoudDelete {
		return fiber.NewError(fiber.StatusForbidden, "You don't have permission to delete this submission")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer tx.Rollback(ctx)

	submissionRepo := s.submissionRepo.WithTx(tx)

	isLatestApprovedDeleted := false

	if submission.Status == constants.StatusTypeApproved {
		projectUUID, err := convert.StringToUUID(submission.ProjectID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid project ID")
		}

		latestApproved, err := s.submissionRepo.GetLatestApprovedSubmission(ctx, projectUUID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to check latest approved submission: "+err.Error())
		}

		if err == nil && latestApproved != nil && latestApproved.ID == submission.ID {
			isLatestApprovedDeleted = true
			prevSubmission, err := s.submissionRepo.GetLatestApprovedSubmissionExcluding(ctx, projectUUID, submissionUUID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					if err := s.clearProjectItems(ctx, tx, projectUUID); err != nil {
						return err
					}
				} else {
					return fiber.NewError(fiber.StatusInternalServerError, "Failed to get previous approved submission: "+err.Error())
				}
			} else if prevSubmission != nil {
				prevCommitUUID, err := convert.StringToUUID(prevSubmission.CommitID)
				if err != nil {
					return fiber.NewError(fiber.StatusBadRequest, "Invalid previous commit ID")
				}

				prevCommit, err := s.commitRepo.GetByID(ctx, prevCommitUUID)
				if err != nil {
					return fiber.NewError(fiber.StatusNotFound, "Previous commit not found")
				}

				var prevSnapshotData request.CommitSnapshot
				err = json.Unmarshal(prevCommit.SnapshotJson, &prevSnapshotData)
				if err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "Failed to parse previous commit snapshot")
				}

				if err := s.applySnapshot(ctx, tx, projectUUID, prevCommitUUID, &prevSnapshotData); err != nil {
					return err
				}
			} else {
				if err := s.clearProjectItems(ctx, tx, projectUUID); err != nil {
					return err
				}
			}
		}
	}

	err = submissionRepo.Delete(ctx, submissionUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete submission")
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to commit transaction: "+err.Error())
	}

	if isLatestApprovedDeleted {
		go func() {
			bgCtx := context.Background()
			_ = s.c.DelByPattern(bgCtx, "entity:search*")
			_ = s.c.DelByPattern(bgCtx, "geometry:search*")
			_ = s.c.DelByPattern(bgCtx, "geometry:search:entity*")
			_ = s.c.DelByPattern(bgCtx, "wiki:search*")
		}()
	}

	_ = s.c.Del(ctx,
		cache.Key("project:id", submission.ProjectID),
		cache.Key("entity:project", submission.ProjectID),
		cache.Key("geometry:project", submission.ProjectID),
		cache.Key("wiki:project", submission.ProjectID),
		cache.Key("battle_replay:project", submission.ProjectID),
	)

	return nil
}

func (s *submissionService) applySnapshot(ctx context.Context, tx pgx.Tx, projectUUID pgtype.UUID, commitUUID pgtype.UUID, snapshotData *request.CommitSnapshot) *fiber.Error {
	entityRepo := s.entityRepo.WithTx(tx)
	geometryRepo := s.geometryRepo.WithTx(tx)
	wikiRepo := s.wikiRepo.WithTx(tx)
	battleReplayRepo := s.battleReplayRepo.WithTx(tx)

	projectIDStr := convert.UUIDToString(projectUUID)
	_ = s.c.Del(ctx,
		cache.Key("entity:project", projectIDStr),
		cache.Key("geometry:project", projectIDStr),
		cache.Key("wiki:project", projectIDStr),
		cache.Key("battle_replay:project", projectIDStr),
	)

	currentEntity, err := entityRepo.GetByProjectID(ctx, projectUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Entity not found: "+err.Error())
	}

	currentGeometry, err := geometryRepo.GetByProjectID(ctx, projectUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Geometry not found: "+err.Error())
	}

	currentWiki, err := wikiRepo.GetByProjectID(ctx, projectUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Wiki not found: "+err.Error())
	}

	currentBattleReplay, err := battleReplayRepo.GetByProjectID(ctx, projectUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Battle replay not found: "+err.Error())
	}

	persistEntityIDs := make(map[string]struct{}, len(snapshotData.Entities))
	for _, item := range snapshotData.Entities {
		persistEntityIDs[item.ID] = struct{}{}
	}
	persistGeometryIDs := make(map[string]struct{}, len(snapshotData.Geometries))
	for _, item := range snapshotData.Geometries {
		persistGeometryIDs[item.ID] = struct{}{}
	}
	persistWikiIDs := make(map[string]struct{}, len(snapshotData.Wikis))
	for _, item := range snapshotData.Wikis {
		persistWikiIDs[item.ID] = struct{}{}
	}
	persistReplayIDs := make(map[string]struct{}, len(snapshotData.Replays))
	for _, item := range snapshotData.Replays {
		persistReplayIDs[item.ID] = struct{}{}
	}

	persistCurrentEntityIDs := make(map[string]struct{}, len(currentEntity))
	for _, item := range currentEntity {
		persistCurrentEntityIDs[item.ID] = struct{}{}
	}
	persistCurrentGeometryIDs := make(map[string]struct{}, len(currentGeometry))
	for _, item := range currentGeometry {
		persistCurrentGeometryIDs[item.ID] = struct{}{}
	}
	persistCurrentWikiIDs := make(map[string]struct{}, len(currentWiki))
	for _, item := range currentWiki {
		persistCurrentWikiIDs[item.ID] = struct{}{}
	}
	persistCurrentReplayIDs := make(map[string]struct{}, len(currentBattleReplay))
	for _, item := range currentBattleReplay {
		persistCurrentReplayIDs[item.ID] = struct{}{}
	}

	listDeleteEntities := make([]pgtype.UUID, 0, len(currentEntity))
	listDeleteWikis := make([]pgtype.UUID, 0, len(currentWiki))
	listDeleteGeometries := make([]pgtype.UUID, 0, len(currentGeometry))
	listDeleteBattleReplays := make([]pgtype.UUID, 0, len(currentBattleReplay))

	for _, e := range currentEntity {
		if _, ok := persistEntityIDs[e.ID]; !ok {
			itemUUID, err := convert.StringToUUID(e.ID)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Invalid entity ID")
			}
			listDeleteEntities = append(listDeleteEntities, itemUUID)
			delete(persistCurrentEntityIDs, e.ID)
		}
	}

	for _, g := range currentGeometry {
		if _, ok := persistGeometryIDs[g.ID]; !ok {
			itemUUID, err := convert.StringToUUID(g.ID)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Invalid geometry ID")
			}
			listDeleteGeometries = append(listDeleteGeometries, itemUUID)
			delete(persistCurrentGeometryIDs, g.ID)
		}
	}

	for _, w := range currentWiki {
		if _, ok := persistWikiIDs[w.ID]; !ok {
			itemUUID, err := convert.StringToUUID(w.ID)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Invalid wiki ID")
			}
			listDeleteWikis = append(listDeleteWikis, itemUUID)
			delete(persistCurrentWikiIDs, w.ID)
		}
	}

	for _, br := range currentBattleReplay {
		if _, ok := persistReplayIDs[br.ID]; !ok {
			itemUUID, err := convert.StringToUUID(br.ID)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Invalid battle replay ID")
			}
			listDeleteBattleReplays = append(listDeleteBattleReplays, itemUUID)
			delete(persistCurrentReplayIDs, br.ID)
		}
	}

	if len(listDeleteEntities) > 0 {
		if err = entityRepo.DeleteByIDs(ctx, listDeleteEntities); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete entities")
		}
	}

	if len(listDeleteGeometries) > 0 {
		if err = geometryRepo.DeleteByIDs(ctx, listDeleteGeometries); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete geometries")
		}
	}

	if len(listDeleteWikis) > 0 {
		if err = wikiRepo.DeleteByIDs(ctx, listDeleteWikis); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete wikis")
		}
	}

	if len(listDeleteBattleReplays) > 0 {
		if err = battleReplayRepo.DeleteByIDs(ctx, listDeleteBattleReplays); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete battle replays")
		}
	}

	refEntityIDs := make([]string, 0, len(snapshotData.Entities))
	for _, e := range snapshotData.Entities {
		if e.Source == "ref" {
			refEntityIDs = append(refEntityIDs, e.ID)
		}
	}

	refEntities, _ := entityRepo.GetByIDs(ctx, refEntityIDs)
	refEntityMap := make(map[string]bool, len(refEntities))
	for _, e := range refEntities {
		refEntityMap[e.ID] = true
	}

	newEntities := make([]*request.EntitySnapshot, 0, len(snapshotData.Entities))
	for i, entity := range snapshotData.Entities {
		if entity.Operation == "delete" {
			continue
		}

		entityUUID, err := convert.StringToUUID(entity.ID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Invalid entity ID")
		}

		if _, ok := persistCurrentEntityIDs[entity.ID]; ok {
			_, err := entityRepo.Update(ctx, sqlc.UpdateEntityParams{
				Name:        convert.StringToText(entity.Name),
				Description: convert.StringToText(entity.Description),
				Slug:        convert.PtrToText(entity.Slug),
				Status:      convert.PtrToInt2(entity.Status),
				TimeStart:   convert.PtrFloat64ToInt4(entity.TimeStart),
				TimeEnd:     convert.PtrFloat64ToInt4(entity.TimeEnd),
				ID:          entityUUID,
			})

			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to update entity: "+err.Error())
			}

			newEntities = append(newEntities, snapshotData.Entities[i])

		} else if entity.Source == "inline" {
			_, err := entityRepo.Create(ctx, sqlc.CreateEntityParams{
				ID:          entityUUID,
				Name:        entity.Name,
				Description: convert.StringToText(entity.Description),
				ProjectID:   projectUUID,
				Slug:        convert.PtrToText(entity.Slug),
				Status:      convert.PtrToInt2(entity.Status),
				TimeStart:   convert.PtrFloat64ToInt4(entity.TimeStart),
				TimeEnd:     convert.PtrFloat64ToInt4(entity.TimeEnd),
			})

			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to create entity: "+err.Error())
			}

			newEntities = append(newEntities, snapshotData.Entities[i])

		} else if entity.Source == "ref" {
			if !refEntityMap[entity.ID] {
				continue
			}
			newEntities = append(newEntities, snapshotData.Entities[i])
		}
	}
	snapshotData.Entities = newEntities

	refGeometryIDs := make([]string, 0, len(snapshotData.Geometries))
	for _, g := range snapshotData.Geometries {
		if g.Source == "ref" {
			refGeometryIDs = append(refGeometryIDs, g.ID)
		}
	}
	refGeometries, _ := geometryRepo.GetByIDs(ctx, refGeometryIDs)
	refGeometryMap := make(map[string]bool, len(refGeometries))
	for _, g := range refGeometries {
		refGeometryMap[g.ID] = true
	}

	validGeometries := make(map[string]bool, len(snapshotData.Geometries))
	for _, g := range snapshotData.Geometries {
		if g.Operation != "delete" {
			validGeometries[g.ID] = true
		}
	}

	newGeometries := make([]*request.GeometrySnapshot, 0, len(snapshotData.Geometries))
	for i, geo := range snapshotData.Geometries {
		if geo.Operation == "delete" {
			continue
		}

		geometryUUID, err := convert.StringToUUID(geo.ID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Invalid geometry ID")
		}

		geoTypeCode := int16(0)
		if geo.Type != "" {
			if n, err := strconv.ParseInt(geo.Type, 10, 16); err == nil {
				geoTypeCode = int16(n)
			}
		}

		if _, ok := persistCurrentGeometryIDs[geo.ID]; ok {
			params := sqlc.UpdateGeometryParams{
				ID:              geometryUUID,
				GeoType:         pgtype.Int2{Int16: geoTypeCode, Valid: true},
				DrawGeometry:    geo.DrawGeometry,
				UpdateBoundWith: pgtype.Bool{Bool: false, Valid: true},
				TimeStart:       convert.PtrFloat64ToInt4(geo.TimeStart),
				TimeEnd:         convert.PtrFloat64ToInt4(geo.TimeEnd),
				ProjectID:       projectUUID,
			}

			if geo.BBox != nil {
				params.UpdateBbox = pgtype.Bool{Bool: true, Valid: true}
				params.MinLng = pgtype.Float8{Float64: geo.BBox.MinLng, Valid: true}
				params.MinLat = pgtype.Float8{Float64: geo.BBox.MinLat, Valid: true}
				params.MaxLng = pgtype.Float8{Float64: geo.BBox.MaxLng, Valid: true}
				params.MaxLat = pgtype.Float8{Float64: geo.BBox.MaxLat, Valid: true}
			}

			_, err := geometryRepo.Update(ctx, params)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to update geometry: "+err.Error())
			}
			newGeometries = append(newGeometries, snapshotData.Geometries[i])

		} else if geo.Source == "inline" {
			params := sqlc.CreateGeometryParams{
				ID:           geometryUUID,
				GeoType:      geoTypeCode,
				DrawGeometry: geo.DrawGeometry,
				BoundWith:    pgtype.UUID{Valid: false},
				TimeStart:    convert.PtrFloat64ToInt4(geo.TimeStart),
				TimeEnd:      convert.PtrFloat64ToInt4(geo.TimeEnd),
				ProjectID:    projectUUID,
			}
			if geo.BBox != nil {
				params.MinLng = geo.BBox.MinLng
				params.MinLat = geo.BBox.MinLat
				params.MaxLng = geo.BBox.MaxLng
				params.MaxLat = geo.BBox.MaxLat
			}

			_, err := geometryRepo.Create(ctx, params)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to create geometry: "+err.Error())
			}
			newGeometries = append(newGeometries, snapshotData.Geometries[i])

		} else if geo.Source == "ref" {
			if !refGeometryMap[geo.ID] {
				continue
			}
			newGeometries = append(newGeometries, snapshotData.Geometries[i])
		}
	}
	snapshotData.Geometries = newGeometries

	for _, geo := range snapshotData.Geometries {
		if geo.Operation == "delete" || geo.Source == "ref" {
			continue
		}

		geometryUUID, err := convert.StringToUUID(geo.ID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Invalid geometry ID")
		}

		var boundWith pgtype.UUID
		if geo.BoundWith != nil && *geo.BoundWith != "" {
			if !validGeometries[*geo.BoundWith] {
				return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Geometry %s references a non-existent or deleted geometry %s as bound_with", geo.ID, *geo.BoundWith))
			}
			if *geo.BoundWith == geo.ID {
				return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Geometry %s cannot be bound to itself", geo.ID))
			}

			boundWith, err = convert.StringToUUID(*geo.BoundWith)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "Invalid bound_with geometry ID")
			}
		}
		params := sqlc.UpdateGeometryParams{
			ID:              geometryUUID,
			UpdateBoundWith: pgtype.Bool{Bool: true, Valid: true},
			BoundWith:       boundWith,
		}
		_, err = geometryRepo.Update(ctx, params)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to update geometry bound_with: "+err.Error())
		}
	}

	refWikiIDs := make([]string, 0, len(snapshotData.Wikis))
	for _, w := range snapshotData.Wikis {
		if w.Source == "ref" {
			refWikiIDs = append(refWikiIDs, w.ID)
		}
	}
	refWikis, _ := wikiRepo.GetByIDs(ctx, refWikiIDs)
	refWikiMap := make(map[string]bool, len(refWikis))
	for _, w := range refWikis {
		refWikiMap[w.ID] = true
	}

	newWikis := make([]*request.WikiSnapshot, 0, len(snapshotData.Wikis))
	for i, wiki := range snapshotData.Wikis {
		if wiki.Operation == "delete" {
			continue
		}

		wikiUUID, err := convert.StringToUUID(wiki.ID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Invalid wiki ID")
		}

		if _, ok := persistCurrentWikiIDs[wiki.ID]; ok {
			_, err := wikiRepo.Update(ctx, sqlc.UpdateWikiParams{
				ID:        wikiUUID,
				Title:     convert.StringToText(wiki.Title),
				Slug:      convert.PtrToText(wiki.Slug),
				ProjectID: projectUUID,
			})
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to update wiki: "+err.Error())
			}

			count, err := wikiRepo.GetContentCountByWikiID(ctx, wikiUUID)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to get wiki content count: "+err.Error())
			}
			versionTitle := fmt.Sprintf("Version %d", count+1)

			var preview pgtype.Text
			if match := blockquoteRegex.FindString(wiki.Doc); match != "" {
				preview = convert.StringToText(match)
			}

			_, err = wikiRepo.CreateContent(ctx, sqlc.CreateWikiContentParams{
				WikiID:  wikiUUID,
				Title:   versionTitle,
				Content: convert.StringToText(wiki.Doc),
				Preview: preview,
			})
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to create wiki content: "+err.Error())
			}

			_ = s.c.Del(ctx, cache.Key("wiki:id", wikiUUID.String()), cache.Key("wiki:slug", *wiki.Slug))

			newWikis = append(newWikis, snapshotData.Wikis[i])

		} else if wiki.Source == "inline" {
			_, err := wikiRepo.Create(ctx, sqlc.CreateWikiParams{
				ID:        wikiUUID,
				Title:     convert.StringToText(wiki.Title),
				Slug:      convert.PtrToText(wiki.Slug),
				ProjectID: projectUUID,
			})
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to create wiki: "+err.Error())
			}

			var preview pgtype.Text
			if match := blockquoteRegex.FindString(wiki.Doc); match != "" {
				preview = convert.StringToText(match)
			}

			_, err = wikiRepo.CreateContent(ctx, sqlc.CreateWikiContentParams{
				WikiID:  wikiUUID,
				Title:   "Version 1",
				Content: convert.StringToText(wiki.Doc),
				Preview: preview,
			})
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to create wiki content: "+err.Error())
			}

			_ = s.c.Del(ctx, cache.Key("wiki:id", wikiUUID.String()), cache.Key("wiki:slug", *wiki.Slug))

			newWikis = append(newWikis, snapshotData.Wikis[i])

		} else if wiki.Source == "ref" {
			if !refWikiMap[wiki.ID] {
				continue
			}
			newWikis = append(newWikis, snapshotData.Wikis[i])
		}
	}
	snapshotData.Wikis = newWikis

	for _, replay := range snapshotData.Replays {
		replayUUID, err := convert.StringToUUID(replay.ID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Invalid battle replay ID")
		}

		geomUUID, err := convert.StringToUUID(replay.GeometryID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Invalid geometry ID in battle replay")
		}

		targetIDs, err := json.Marshal(replay.TargetGeometryIDs)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to marshal target geometry IDs")
		}

		if _, ok := persistCurrentReplayIDs[replay.ID]; ok {
			_, err := battleReplayRepo.Update(ctx, sqlc.UpdateBattleReplayParams{
				ID:                replayUUID,
				GeometryID:        geomUUID,
				TargetGeometryIds: targetIDs,
				Detail:            replay.Detail,
			})
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to update battle replay: "+err.Error())
			}
		} else {
			_, err := battleReplayRepo.Create(ctx, sqlc.CreateBattleReplayParams{
				ID:                replayUUID,
				GeometryID:        geomUUID,
				ProjectID:         projectUUID,
				TargetGeometryIds: targetIDs,
				Detail:            replay.Detail,
			})
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to create battle replay: "+err.Error())
			}
		}
	}

	err = geometryRepo.DeleteEntityGeometriesByProjectID(ctx, projectUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete geometry entity: "+err.Error())
	}
	err = wikiRepo.DeleteEntityWikisByProjectID(ctx, projectUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete wiki entity: "+err.Error())
	}

	validEntities := make(map[string]bool, len(snapshotData.Entities))
	for _, e := range snapshotData.Entities {
		validEntities[e.ID] = true
	}
	validWikis := make(map[string]bool, len(snapshotData.Wikis))
	for _, w := range snapshotData.Wikis {
		validWikis[w.ID] = true
	}

	if len(snapshotData.GeometryEntity) > 0 {
		geomLinks := make(map[string][]pgtype.UUID, len(snapshotData.GeometryEntity))
		for _, link := range snapshotData.GeometryEntity {
			if link.Operation == "delete" {
				continue
			}
			if !validEntities[link.EntityID] || !validGeometries[link.GeometryID] {
				continue
			}
			gID, _ := convert.StringToUUID(link.GeometryID)
			geomLinks[link.EntityID] = append(geomLinks[link.EntityID], gID)
		}

		for eIDStr, gIDs := range geomLinks {
			eID, _ := convert.StringToUUID(eIDStr)
			err = geometryRepo.CreateEntityGeometries(ctx, sqlc.CreateEntityGeometriesParams{
				EntityID:    eID,
				GeometryIds: gIDs,
				ProjectID:   projectUUID,
			})
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to create geometry entity: "+err.Error())
			}
		}
	}

	if len(snapshotData.EntityWiki) > 0 {
		wikiLinks := make(map[string][]pgtype.UUID, len(snapshotData.EntityWiki))
		for _, link := range snapshotData.EntityWiki {
			if link.Operation == "delete" || (link.IsDeleted != nil && *link.IsDeleted == 1) {
				continue
			}
			if !validEntities[link.EntityID] || !validWikis[link.WikiID] {
				continue
			}
			wID, _ := convert.StringToUUID(link.WikiID)
			wikiLinks[link.EntityID] = append(wikiLinks[link.EntityID], wID)
		}

		for eIDStr, wIDs := range wikiLinks {
			eID, _ := convert.StringToUUID(eIDStr)
			err = wikiRepo.CreateEntityWikis(ctx, sqlc.CreateEntityWikisParams{
				EntityID:  eID,
				WikiIds:   wIDs,
				ProjectID: projectUUID,
			})
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to create wiki entity: "+err.Error())
			}
		}
	}

	wikiDeleteIDs := make([]string, 0, len(listDeleteWikis)+len(snapshotData.Wikis))
	entityDeleteIDs := make([]string, 0, len(listDeleteEntities)+len(snapshotData.Entities))

	for _, id := range listDeleteWikis {
		wikiDeleteIDs = append(wikiDeleteIDs, convert.UUIDToString(id))
	}
	for _, id := range listDeleteEntities {
		entityDeleteIDs = append(entityDeleteIDs, convert.UUIDToString(id))
	}

	for _, wiki := range snapshotData.Wikis {
		if wiki.Operation == "delete" {
			wikiDeleteIDs = append(wikiDeleteIDs, wiki.ID)
		}
	}
	for _, entity := range snapshotData.Entities {
		if entity.Operation == "delete" {
			entityDeleteIDs = append(entityDeleteIDs, entity.ID)
		}
	}

	ragTask := models.RagIndexTask{
		ProjectID:       convert.UUIDToString(projectUUID),
		DeleteWikiIDs:   wikiDeleteIDs,
		DeleteEntityIDs: entityDeleteIDs,
		Wikis:           make([]*models.RagWikiItem, 0, len(snapshotData.Wikis)),
		Entities:        make([]*models.RagEntityItem, 0, len(snapshotData.Entities)),
	}

	for _, wiki := range snapshotData.Wikis {
		ragTask.Wikis = append(ragTask.Wikis, &models.RagWikiItem{
			ID:     wiki.ID,
			Title:  wiki.Title,
			Doc:    wiki.Doc,
			Source: wiki.Source,
		})
	}
	for _, entity := range snapshotData.Entities {
		ragTask.Entities = append(ragTask.Entities, &models.RagEntityItem{
			ID:          entity.ID,
			Name:        entity.Name,
			Description: entity.Description,
			Source:      entity.Source,
		})
	}

	if err := s.c.PublishTask(ctx, constants.StreamRagName, constants.TaskTypeRagIndexSubmission, ragTask); err != nil {
		log.Error().Err(err).Str("project_id", convert.UUIDToString(projectUUID)).Msg("Failed to publish RAG index task")
	}

	return nil
}

func (s *submissionService) clearProjectItems(ctx context.Context, tx pgx.Tx, projectUUID pgtype.UUID) *fiber.Error {
	entityRepo := s.entityRepo.WithTx(tx)
	geometryRepo := s.geometryRepo.WithTx(tx)
	wikiRepo := s.wikiRepo.WithTx(tx)
	battleReplayRepo := s.battleReplayRepo.WithTx(tx)

	projectIDStr := convert.UUIDToString(projectUUID)
	_ = s.c.Del(ctx,
		cache.Key("entity:project", projectIDStr),
		cache.Key("geometry:project", projectIDStr),
		cache.Key("wiki:project", projectIDStr),
		cache.Key("battle_replay:project", projectIDStr),
	)

	currentEntity, _ := entityRepo.GetByProjectID(ctx, projectUUID)
	currentGeometry, _ := geometryRepo.GetByProjectID(ctx, projectUUID)
	currentWiki, _ := wikiRepo.GetByProjectID(ctx, projectUUID)
	currentBattleReplay, _ := battleReplayRepo.GetByProjectID(ctx, projectUUID)

	entityIDs := make([]pgtype.UUID, 0, len(currentEntity))
	for _, e := range currentEntity {
		id, err := convert.StringToUUID(e.ID)
		if err == nil {
			entityIDs = append(entityIDs, id)
		}
	}
	geometryIDs := make([]pgtype.UUID, 0, len(currentGeometry))
	for _, g := range currentGeometry {
		id, err := convert.StringToUUID(g.ID)
		if err == nil {
			geometryIDs = append(geometryIDs, id)
		}
	}
	wikiIDs := make([]pgtype.UUID, 0, len(currentWiki))
	for _, w := range currentWiki {
		id, err := convert.StringToUUID(w.ID)
		if err == nil {
			wikiIDs = append(wikiIDs, id)
		}
	}
	replayIDs := make([]pgtype.UUID, 0, len(currentBattleReplay))
	for _, br := range currentBattleReplay {
		id, err := convert.StringToUUID(br.ID)
		if err == nil {
			replayIDs = append(replayIDs, id)
		}
	}

	if len(entityIDs) > 0 {
		_ = entityRepo.DeleteByIDs(ctx, entityIDs)
		for _, e := range currentEntity {
			_ = s.c.Del(ctx, cache.Key("entity:slug", e.Slug))
		}
	}
	if len(geometryIDs) > 0 {
		_ = geometryRepo.DeleteByIDs(ctx, geometryIDs)
	}
	if len(wikiIDs) > 0 {
		_ = wikiRepo.DeleteByIDs(ctx, wikiIDs)
		for _, w := range currentWiki {
			_ = s.c.Del(ctx, cache.Key("wiki:slug", w.Slug))
		}
	}
	if len(replayIDs) > 0 {
		_ = battleReplayRepo.DeleteByIDs(ctx, replayIDs)
	}

	_ = geometryRepo.DeleteEntityGeometriesByProjectID(ctx, projectUUID)
	_ = wikiRepo.DeleteEntityWikisByProjectID(ctx, projectUUID)

	entityDeleteIDs := make([]string, 0, len(currentEntity))
	for _, e := range currentEntity {
		entityDeleteIDs = append(entityDeleteIDs, e.ID)
	}
	wikiDeleteIDs := make([]string, 0, len(currentWiki))
	for _, w := range currentWiki {
		wikiDeleteIDs = append(wikiDeleteIDs, w.ID)
	}
	if len(entityDeleteIDs) > 0 || len(wikiDeleteIDs) > 0 {
		ragTask := models.RagIndexTask{
			ProjectID:       convert.UUIDToString(projectUUID),
			DeleteWikiIDs:   wikiDeleteIDs,
			DeleteEntityIDs: entityDeleteIDs,
		}
		_ = s.c.PublishTask(ctx, constants.StreamRagName, constants.TaskTypeRagIndexSubmission, ragTask)
	}

	return nil
}
