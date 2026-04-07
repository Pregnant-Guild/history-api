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
	"history-api/pkg/cache"
	"history-api/pkg/constants"
	"history-api/pkg/convert"
	"history-api/pkg/storage"
	"io"
	"mime/multipart"
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type MediaService interface {
	GetMediaByID(ctx context.Context, mediaId string) (*response.MediaResponse, error)
	GetMediaByUserID(ctx context.Context, userId string) ([]*response.MediaResponse, error)
	SearchMedia(ctx context.Context, dto *request.SearchMediaDto) (*response.PaginatedResponse, error)
	DeleteMedia(ctx context.Context, claims *response.JWTClaims, mediaId string) error
	UploadServerSide(ctx context.Context, userId string, fileHeader *multipart.FileHeader) (*response.MediaResponse, error)
	GeneratePresignedURL(ctx context.Context, userId string, dto *request.PreSignedDto) (*response.PreSignedResponse, error)
	PreSignedCompleted(ctx context.Context, userId string, dto *request.PreSignedCompleteDto) (*response.MediaResponse, error)
}

type mediaService struct {
	mediaRepo repositories.MediaRepository
	tokenRepo repositories.TokenRepository
	s         storage.Storage
	c         cache.Cache
}

func NewMediaService(
	mediaRepo repositories.MediaRepository,
	tokenRepo repositories.TokenRepository,
	s storage.Storage,
	c cache.Cache,
) MediaService {
	return &mediaService{
		mediaRepo: mediaRepo,
		tokenRepo: tokenRepo,
		s:         s,
		c:         c,
	}
}

func (m *mediaService) DeleteMedia(ctx context.Context, claims *response.JWTClaims, mediaId string) error {
	mediaIdUUID, err := convert.StringToUUID(mediaId)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	media, err := m.mediaRepo.GetByID(ctx, mediaIdUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	shoudDelete := false
	if slices.Contains(claims.Roles, constants.ADMIN) || slices.Contains(claims.Roles, constants.MOD) || media.UserID == claims.UId {
		shoudDelete = true
	}

	if !shoudDelete {
		return fiber.NewError(fiber.StatusForbidden, "You don't have permission to delete this media")
	}

	err = m.mediaRepo.Delete(ctx, mediaIdUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	m.c.PublishTask(ctx, constants.StreamStorageName, constants.TaskTypeDeleteMedia, media.ToStorageEntity())

	return nil
}

func (m *mediaService) GetMediaByID(ctx context.Context, id string) (*response.MediaResponse, error) {
	mediaId, err := convert.StringToUUID(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	media, err := m.mediaRepo.GetByID(ctx, mediaId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return media.ToResponse(), nil
}

func (m *mediaService) GetMediaByUserID(ctx context.Context, id string) ([]*response.MediaResponse, error) {
	userId, err := convert.StringToUUID(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	medias, err := m.mediaRepo.GetByUserID(ctx, userId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return models.MediaEntitiesToResponse(medias), nil
}

func (m *mediaService) SearchMedia(ctx context.Context, dto *request.SearchMediaDto) (*response.PaginatedResponse, error) {
	arg := sqlc.SearchMediasParams{
		Limit: int32(dto.Limit + 1),
	}
	
	if dto.Sort != "" {
		arg.Sort = pgtype.Text{String: dto.Sort, Valid: true}
	} else {
		arg.Sort = pgtype.Text{String: "id", Valid: true}
	}

	if dto.Order != "" {
		arg.Order = pgtype.Text{String: dto.Order, Valid: true}
	} else {
		arg.Order = pgtype.Text{String: "asc", Valid: true}
	}

	if dto.Cursor != "" {
		pgID, err := convert.StringToUUID(dto.Cursor)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid cursor format")
		}
		arg.Cursor = pgID
	}

	if dto.Search != "" {
		arg.SearchText = pgtype.Text{String: dto.Search, Valid: true}
	}

	rows, err := m.mediaRepo.Search(ctx, arg)
	if err != nil {
		return nil, err
	}

	hasMore := false
	var nextCursor string

	if len(rows) > dto.Limit {
		hasMore = true
		nextCursor = rows[dto.Limit-1].ID
		rows = rows[:dto.Limit]
	}

	res := &response.PaginatedResponse{
		Data:    rows,
		Status:  true,
		Message: "",
	}
	res.Pagination.HasMore = hasMore
	res.Pagination.NextCursor = nextCursor

	return res, nil
}

func (m *mediaService) UploadServerSide(ctx context.Context, userId string, fileHeader *multipart.FileHeader) (*response.MediaResponse, error) {
	userIdUUID, err := convert.StringToUUID(userId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Cannot open file")
	}
	defer file.Close()
	var reader io.Reader = file
	fileExt := filepath.Ext(fileHeader.Filename)
	contentType := fileHeader.Header.Get("Content-Type")
	mid, err := uuid.NewV7()
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to generate media ID")
	}
	newFileName := mid.String() + fileExt
	originalName := fileHeader.Filename
	encodedName := url.QueryEscape(originalName)

	dispositionType := "attachment"
	if strings.HasPrefix(contentType, "image/") || contentType == "application/pdf" {
		dispositionType = "inline"
	}

	contentDisposition := fmt.Sprintf("%s; filename=\"%s\"; filename*=UTF-8''%s",
		dispositionType,
		"file"+fileExt,
		encodedName,
	)

	metadata := map[string]string{
		"original-name": encodedName,
	}

	mdByte, err := json.Marshal(metadata)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to encode metadata")
	}

	err = m.s.Upload(ctx, newFileName, reader, fileHeader.Size, storage.UploadOptions{
		ContentType:        contentType,
		ContentDisposition: contentDisposition,
		Metadata:           metadata,
	})
	if err != nil {
		log.Err(err).Msg("Failed to upload file to storage")
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to upload file")
	}

	media, err := m.mediaRepo.Create(ctx, sqlc.CreateMediaParams{
		UserID:       userIdUUID,
		StorageKey:   newFileName,
		OriginalName: originalName,
		MimeType:     contentType,
		Size:         fileHeader.Size,
		FileMetadata: mdByte,
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return media.ToResponse(), nil
}

func (m *mediaService) GeneratePresignedURL(ctx context.Context, userId string, dto *request.PreSignedDto) (*response.PreSignedResponse, error) {
	fileExt := filepath.Ext(dto.FileName)
	mid, err := uuid.NewV7()
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to generate media ID")
	}
	newFileName := mid.String() + fileExt
	encodedName := url.QueryEscape(dto.FileName)

	dispositionType := "attachment"
	if dto.ContentType == "application/pdf" || (len(dto.ContentType) > 6 && dto.ContentType[:6] == "image/") {
		dispositionType = "inline"
	}

	contentDisposition := fmt.Sprintf("%s; filename=\"%s\"; filename*=UTF-8''%s",
		dispositionType, "file"+fileExt, encodedName)

	metadata := map[string]string{
		"original-name": encodedName,
	}
	mdByte, err := json.Marshal(metadata)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to encode metadata")
	}

	presignedURL, err := m.s.PresignUpload(ctx, newFileName, constants.PreSignedURLDuration, storage.UploadOptions{
		ContentType:        dto.ContentType,
		ContentDisposition: contentDisposition,
		Metadata:           metadata,
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to generate presigned URL")
	}

	tokenId := uuid.New().String()
	err = m.tokenRepo.CreateUploadToken(
		ctx,
		userId,
		&models.TokenUploadEntity{
			ID:           tokenId,
			UserID:       userId,
			StorageKey:   newFileName,
			OriginalName: dto.FileName,
			MimeType:     dto.ContentType,
			Size:         dto.Size,
			FileMetadata: mdByte,
		},
	)

	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Internal Server Error")
	}

	return &response.PreSignedResponse{
		TokenID:    tokenId,
		UploadUrl:  presignedURL,
		StorageKey: newFileName,
		SignedHeaders: map[string]string{
			"x-amz-meta-original-name": encodedName,
			"Content-Disposition":      contentDisposition,
		},
	}, nil
}

func (m *mediaService) PreSignedCompleted(ctx context.Context, userId string, dto *request.PreSignedCompleteDto) (*response.MediaResponse, error) {
	token, err := m.tokenRepo.GetUploadToken(ctx, userId, dto.TokenID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to get upload token")
	}
	if token == nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid or expired token")
	}
	userIdUUID, err := convert.StringToUUID(userId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	err = m.s.Move(
		ctx,
		&storage.MoveOptions{
			Bucket: m.s.GetTempBucket(),
			Key:    token.StorageKey,
		},
		&storage.MoveOptions{
			Bucket: m.s.GetMainBucket(),
			Key:    token.StorageKey,
		},
	)

	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to move file to final destination")
	}

	media, err := m.mediaRepo.Create(ctx, sqlc.CreateMediaParams{
		UserID:       userIdUUID,
		StorageKey:   token.StorageKey,
		OriginalName: token.OriginalName,
		MimeType:     token.MimeType,
		Size:         token.Size,
		FileMetadata: token.FileMetadata,
	})
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create media record")
	}

	_ = m.tokenRepo.DeleteUploadToken(ctx, userId, dto.TokenID)

	return media.ToResponse(), nil
}
