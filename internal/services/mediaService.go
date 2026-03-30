package services

import (
	"context"
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/repositories"
	"history-api/pkg/storage"
	"mime/multipart"
)

type MediaService interface {
	GetMediaByID(ctx context.Context, mediaId string) (*response.MediaResponse, error)
	GetMediaByUserID(ctx context.Context, userId string) ([]*response.MediaResponse, error)
	SearchMedia(ctx context.Context, dto *request.SearchMediaDto) (*response.PaginatedResponse, error)
	DeleteMedia(ctx context.Context, mediaId string) error
	GetMediaByTarget(ctx context.Context, targetType string, targetId string) ([]*response.MediaResponse, error)
	UploadServerSide(ctx context.Context, userId string, fileHeader *multipart.FileHeader) (*response.MediaResponse, error)
	GeneratePresignedURL(ctx context.Context, userId string, dto *request.PreSignedDto) (*response.PreSignedResponse, error)
	PreSignedCompleted(ctx context.Context, userId string, dto *request.PreSignedCompleteDto) (*response.MediaResponse, error)
}

type mediaService struct {
	mediaRepo repositories.MediaRepository
	s         storage.Storage
}

func NewMediaService(
	mediaRepo repositories.MediaRepository,
	s storage.Storage,
) MediaService {
	return &mediaService{
		mediaRepo: mediaRepo,
		s:         s,
	}
}

// DeleteMedia implements [MediaService].
func (m *mediaService) DeleteMedia(ctx context.Context, mediaId string) error {
	panic("unimplemented")
}

// GeneratePresignedURL implements [MediaService].
func (m *mediaService) GeneratePresignedURL(ctx context.Context, userId string, dto *request.PreSignedDto) (*response.PreSignedResponse, error) {
	panic("unimplemented")
}

// GetMediaByID implements [MediaService].
func (m *mediaService) GetMediaByID(ctx context.Context, mediaId string) (*response.MediaResponse, error) {
	panic("unimplemented")
}

// GetMediaByTarget implements [MediaService].
func (m *mediaService) GetMediaByTarget(ctx context.Context, targetType string, targetId string) ([]*response.MediaResponse, error) {
	panic("unimplemented")
}

// GetMediaByUserID implements [MediaService].
func (m *mediaService) GetMediaByUserID(ctx context.Context, userId string) ([]*response.MediaResponse, error) {
	panic("unimplemented")
}

// PreSignedCompleted implements [MediaService].
func (m *mediaService) PreSignedCompleted(ctx context.Context, userId string, dto *request.PreSignedCompleteDto) (*response.MediaResponse, error) {
	panic("unimplemented")
}

// SearchMedia implements [MediaService].
func (m *mediaService) SearchMedia(ctx context.Context, dto *request.SearchMediaDto) (*response.PaginatedResponse, error) {
	panic("unimplemented")
}

// UploadServerSide implements [MediaService].
func (m *mediaService) UploadServerSide(ctx context.Context, userId string, fileHeader *multipart.FileHeader) (*response.MediaResponse, error) {
	panic("unimplemented")
}

