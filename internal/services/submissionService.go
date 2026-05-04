package services

import (
	"context"
	"encoding/json"
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
	"slices"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

type SubmissionService interface {
	CreateSubmission(ctx context.Context, userID string, dto *request.CreateSubmissionDto) (*response.SubmissionResponse, *fiber.Error)
	UpdateSubmissionStatus(ctx context.Context, reviewerID string, submissionID string, dto *request.UpdateSubmissionStatusDto) (*response.SubmissionResponse, *fiber.Error)
	GetSubmissionByID(ctx context.Context, id string) (*response.SubmissionResponse, *fiber.Error)
	SearchSubmissions(ctx context.Context, dto *request.SearchSubmissionDto) (*response.PaginatedResponse, *fiber.Error)
	DeleteSubmission(ctx context.Context, userID string, id string, claims *response.JWTClaims) *fiber.Error
}

type submissionService struct {
	submissionRepo repositories.SubmissionRepository
	projectRepo    repositories.ProjectRepository
	commitRepo     repositories.CommitRepository
	userRepo       repositories.UserRepository
	wikiRepo       repositories.WikiRepository
	geometryRepo   repositories.GeometryRepository
	entityRepo     repositories.EntityRepository
	ragRepo        repositories.RagRepository
	ragUtils       *ai.RagUtils
	db             *pgxpool.Pool
	c              cache.Cache
}

func NewSubmissionService(
	submissionRepo repositories.SubmissionRepository,
	projectRepo repositories.ProjectRepository,
	commitRepo repositories.CommitRepository,
	userRepo repositories.UserRepository,
	wikiRepo repositories.WikiRepository,
	geometryRepo repositories.GeometryRepository,
	entityRepo repositories.EntityRepository,
	ragRepo repositories.RagRepository,
	ragUtils *ai.RagUtils,
	db *pgxpool.Pool,
	c cache.Cache,
) SubmissionService {
	return &submissionService{
		submissionRepo: submissionRepo,
		projectRepo:    projectRepo,
		commitRepo:     commitRepo,
		userRepo:       userRepo,
		wikiRepo:       wikiRepo,
		geometryRepo:   geometryRepo,
		entityRepo:     entityRepo,
		ragRepo:        ragRepo,
		ragUtils:       ragUtils,
		db:             db,
		c:              c,
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

	return submission.ToResponse(), nil
}

func (s *submissionService) UpdateSubmissionStatus(ctx context.Context, reviewerID string, submissionID string, dto *request.UpdateSubmissionStatusDto) (*response.SubmissionResponse, *fiber.Error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer tx.Rollback(ctx)

	submissionRepo := s.submissionRepo.WithTx(tx)
	commitRepo := s.commitRepo.WithTx(tx)
	entityRepo := s.entityRepo.WithTx(tx)
	geometryRepo := s.geometryRepo.WithTx(tx)
	wikiRepo := s.wikiRepo.WithTx(tx)
	ragRepo := s.ragRepo.WithTx(tx)

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

	listDeleteEntities := make([]pgtype.UUID, 0)
	listDeleteWikis := make([]pgtype.UUID, 0)
	listDeleteGeometries := make([]pgtype.UUID, 0)
	var snapshotData request.CommitSnapshot
	if status == constants.StatusTypeApproved {
		err = json.Unmarshal(commit.SnapshotJson, &snapshotData)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to parse commit snapshot")
		}

		projectUUID, err := convert.StringToUUID(commit.ProjectID)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid project ID")
		}
		currentEntity, err := s.entityRepo.GetByProjectID(ctx, projectUUID)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusNotFound, "Entity not found")
		}

		currentGeometry, err := s.geometryRepo.GetByProjectID(ctx, projectUUID)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusNotFound, "Geometry not found")
		}

		currentWiki, err := s.wikiRepo.GetByProjectID(ctx, projectUUID)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusNotFound, "Wiki not found")
		}

		persistItemIDs := make(map[string]struct{})
		for _, item := range snapshotData.Entities {
			persistItemIDs[item.ID] = struct{}{}
		}
		for _, item := range snapshotData.Geometries {
			persistItemIDs[item.ID] = struct{}{}
		}
		for _, item := range snapshotData.Wikis {
			persistItemIDs[item.ID] = struct{}{}
		}

		persistCurrentItemIDs := make(map[string]struct{})
		for _, item := range currentEntity {
			persistCurrentItemIDs[item.ID] = struct{}{}
		}
		for _, item := range currentGeometry {
			persistCurrentItemIDs[item.ID] = struct{}{}
		}
		for _, item := range currentWiki {
			persistCurrentItemIDs[item.ID] = struct{}{}
		}

		for _, e := range currentEntity {
			if _, ok := persistItemIDs[e.ID]; !ok {
				itemUUID, err := convert.StringToUUID(e.ID)
				if err != nil {
					return nil, fiber.NewError(fiber.StatusInternalServerError, "Invalid entity ID")
				}
				listDeleteEntities = append(listDeleteEntities, itemUUID)
				delete(persistCurrentItemIDs, e.ID)
			}
		}

		for _, g := range currentGeometry {
			if _, ok := persistItemIDs[g.ID]; !ok {
				itemUUID, err := convert.StringToUUID(g.ID)
				if err != nil {
					return nil, fiber.NewError(fiber.StatusInternalServerError, "Invalid geometry ID")
				}
				listDeleteGeometries = append(listDeleteGeometries, itemUUID)
				delete(persistCurrentItemIDs, g.ID)
			}
		}

		for _, w := range currentWiki {
			if _, ok := persistItemIDs[w.ID]; !ok {
				itemUUID, err := convert.StringToUUID(w.ID)
				if err != nil {
					return nil, fiber.NewError(fiber.StatusInternalServerError, "Invalid wiki ID")
				}
				listDeleteWikis = append(listDeleteWikis, itemUUID)
				delete(persistCurrentItemIDs, w.ID)
			}
		}

		if len(listDeleteEntities) > 0 {
			if err = entityRepo.DeleteByIDs(ctx, listDeleteEntities); err != nil {
				return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to delete entities")
			}
		}

		if len(listDeleteGeometries) > 0 {
			if err = geometryRepo.DeleteByIDs(ctx, listDeleteGeometries); err != nil {
				return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to delete geometries")
			}
		}

		if len(listDeleteWikis) > 0 {
			if err = wikiRepo.DeleteByIDs(ctx, listDeleteWikis); err != nil {
				return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to delete wikis")
			}
		}

		refEntityIDs := []string{}
		for _, e := range snapshotData.Entities {
			if e.Source == "ref" {
				refEntityIDs = append(refEntityIDs, e.ID)
			}
		}

		refEntities, _ := s.entityRepo.GetByIDs(ctx, refEntityIDs)
		refEntityMap := make(map[string]bool)
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
				return nil, fiber.NewError(fiber.StatusInternalServerError, "Invalid entity ID")
			}

			if _, ok := persistCurrentItemIDs[entity.ID]; ok {
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
					return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to update entity: "+entity.ID)
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
					return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create entity: "+entity.ID)
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

		refGeometryIDs := []string{}
		for _, g := range snapshotData.Geometries {
			if g.Source == "ref" {
				refGeometryIDs = append(refGeometryIDs, g.ID)
			}
		}
		refGeometries, _ := s.geometryRepo.GetByIDs(ctx, refGeometryIDs)
		refGeometryMap := make(map[string]bool)
		for _, g := range refGeometries {
			refGeometryMap[g.ID] = true
		}

		newGeometries := make([]*request.GeometrySnapshot, 0, len(snapshotData.Geometries))
		for i, geo := range snapshotData.Geometries {
			if geo.Operation == "delete" {
				continue
			}

			geometryUUID, err := convert.StringToUUID(geo.ID)
			if err != nil {
				return nil, fiber.NewError(fiber.StatusInternalServerError, "Invalid geometry ID")
			}

			binding, _ := json.Marshal(geo.Binding)

			if _, ok := persistCurrentItemIDs[geo.ID]; ok {
				params := sqlc.UpdateGeometryParams{
					ID:           geometryUUID,
					GeoType:      pgtype.Int2{Int16: constants.ParseGeoTypeText(geo.Type).Int16(), Valid: true},
					DrawGeometry: geo.DrawGeometry,
					Binding:      binding,
					TimeStart:    convert.PtrFloat64ToInt4(geo.TimeStart),
					TimeEnd:      convert.PtrFloat64ToInt4(geo.TimeEnd),
					ProjectID:    projectUUID,
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
					return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to update geometry: "+geo.ID)
				}
				newGeometries = append(newGeometries, snapshotData.Geometries[i])

			} else if geo.Source == "inline" {
				params := sqlc.CreateGeometryParams{
					ID:           geometryUUID,
					GeoType:      constants.ParseGeoTypeText(geo.Type).Int16(),
					DrawGeometry: geo.DrawGeometry,
					Binding:      binding,
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
					return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create geometry: "+geo.ID)
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

		refWikiIDs := []string{}
		for _, w := range snapshotData.Wikis {
			if w.Source == "ref" {
				refWikiIDs = append(refWikiIDs, w.ID)
			}
		}
		refWikis, _ := s.wikiRepo.GetByIDs(ctx, refWikiIDs)
		refWikiMap := make(map[string]bool)
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
				return nil, fiber.NewError(fiber.StatusInternalServerError, "Invalid wiki ID")
			}

			if _, ok := persistCurrentItemIDs[wiki.ID]; ok {
				_, err := wikiRepo.Update(ctx, sqlc.UpdateWikiParams{
					ID:        wikiUUID,
					Title:     convert.StringToText(wiki.Title),
					Content:   convert.StringToText(wiki.Doc),
					ProjectID: projectUUID,
				})
				if err != nil {
					return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to update wiki: "+wiki.ID)
				}
				newWikis = append(newWikis, snapshotData.Wikis[i])

			} else if wiki.Source == "inline" {
				_, err := wikiRepo.Create(ctx, sqlc.CreateWikiParams{
					ID:        wikiUUID,
					Title:     convert.StringToText(wiki.Title),
					Content:   convert.StringToText(wiki.Doc),
					ProjectID: projectUUID,
				})
				if err != nil {
					return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create wiki: "+wiki.ID)
				}
				newWikis = append(newWikis, snapshotData.Wikis[i])

			} else if wiki.Source == "ref" {
				if !refWikiMap[wiki.ID] {
					continue
				}
				newWikis = append(newWikis, snapshotData.Wikis[i])
			}
		}
		snapshotData.Wikis = newWikis

		err = geometryRepo.DeleteEntityGeometriesByProjectID(ctx, projectUUID)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to delete geometry entity: "+err.Error())
		}
		err = wikiRepo.DeleteEntityWikisByProjectID(ctx, projectUUID)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to delete wiki entity: "+err.Error())
		}

		validEntities := make(map[string]bool)
		for _, e := range snapshotData.Entities {
			validEntities[e.ID] = true
		}
		validGeometries := make(map[string]bool)
		for _, g := range snapshotData.Geometries {
			validGeometries[g.ID] = true
		}
		validWikis := make(map[string]bool)
		for _, w := range snapshotData.Wikis {
			validWikis[w.ID] = true
		}

		if len(snapshotData.GeometryEntity) > 0 {
			geomLinks := make(map[string][]pgtype.UUID)
			for _, link := range snapshotData.GeometryEntity {
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
					return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create geometry entity: "+err.Error())
				}
			}
		}

		if len(snapshotData.EntityWiki) > 0 {
			wikiLinks := make(map[string][]pgtype.UUID)
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
					return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create wiki entity: "+err.Error())
				}
			}
		}

		newSnapshot, err := json.Marshal(snapshotData)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to marshal snapshot")
		}
		_, err = commitRepo.UpdateSnapshot(ctx, commitUUID, newSnapshot)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to update snapshot: "+err.Error())
		}
	}

	if status == constants.StatusTypeApproved {
		wikiDeleteIDs := make([]string, 0)
		entityDeleteIDs := make([]string, 0)

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

		_ = ragRepo.DeleteBySourceIDs(ctx, "wiki", wikiDeleteIDs)
		_ = ragRepo.DeleteBySourceIDs(ctx, "entity", entityDeleteIDs)

		for _, wiki := range snapshotData.Wikis {
			if wiki.Source == "inline" {
				cleanText := s.ragUtils.StripHTML(wiki.Title + "\n" + wiki.Doc)
				chunks, vectors, err := s.ragUtils.PrepareChunks(ctx, cleanText)
				if err == nil {
					_ = ragRepo.DeleteBySourceIDs(ctx, "wiki", []string{wiki.ID})
					for i, chunk := range chunks {
						_ = ragRepo.SaveChunk(ctx, "wiki", wiki.ID, commit.ProjectID, i, chunk, vectors[i])
					}
				}
			}
		}

		for _, entity := range snapshotData.Entities {
			if entity.Source == "inline" {
				cleanText := s.ragUtils.StripHTML(entity.Name + "\n" + entity.Description)
				chunks, vectors, err := s.ragUtils.PrepareChunks(ctx, cleanText)
				if err == nil {
					_ = ragRepo.DeleteBySourceIDs(ctx, "entity", []string{entity.ID})
					for i, chunk := range chunks {
						_ = ragRepo.SaveChunk(ctx, "entity", entity.ID, commit.ProjectID, i, chunk, vectors[i])
					}
				}
			}
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
			_ = s.c.DelByPattern(bgCtx, "wiki:search*")
		}()
	}

	_ = s.c.Del(ctx, fmt.Sprintf("project:id:%s", submission.ProjectID))

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
		for _, id := range dto.Statuses {
			if u := constants.ParseStatusTypeText(id); u != constants.StatusTypeUnknown {
				arg.Statuses = append(arg.Statuses, u.Int16())
			}
		}
	}

	if len(dto.UserIDs) > 0 {
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

	err = s.submissionRepo.Delete(ctx, submissionUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete submission")
	}

	return nil
}
