package models

import (
	"encoding/json"
	"history-api/internal/dtos/response"
	"history-api/pkg/constants"
	"time"
)

type UserVerificationEntity struct {
	ID         string               `json:"id"`
	UserID     string               `json:"user_id"`
	VerifyType string               `json:"verify_type"`
	Content    string               `json:"content"`
	IsDeleted  bool                 `json:"is_deleted"`
	Status     constants.VerifyType `json:"status"`
	ReviewedBy *string              `json:"reviewed_by"`
	ReviewedAt *time.Time           `json:"reviewed_at"`
	CreatedAt  time.Time            `json:"created_at"`
	Media      []*MediaSimpleEntity `json:"media"`
}

func (u *UserVerificationEntity) ParseMedia(data []byte) error {
	if len(data) == 0 {
		u.Media = []*MediaSimpleEntity{}
		return nil
	}
	return json.Unmarshal(data, &u.Media)
}

func (u *UserVerificationEntity) ToResponse() *response.UserVerificationResponse {
	mediaResponses := make([]*response.MediaSimpleResponse, 0)
	for _, m := range u.Media {
		if m != nil {
			mediaResponses = append(mediaResponses, &response.MediaSimpleResponse{
				ID:           m.ID,
				StorageKey:   m.StorageKey,
				OriginalName: m.OriginalName,
				MimeType:     m.MimeType,
				Size:         m.Size,
				FileMetadata: m.FileMetadata,
				CreatedAt:    m.CreatedAt,
			})
		}
	}

	res := &response.UserVerificationResponse{
		ID:         u.ID,
		UserID:     u.UserID,
		VerifyType: u.VerifyType,
		Content:    u.Content,
		Status:     u.Status.String(),
		CreatedAt:  u.CreatedAt,
		Medias:     mediaResponses,
	}

	if u.ReviewedBy != nil {
		res.ReviewedBy = u.ReviewedBy
	}

	if u.ReviewedAt != nil {
		res.ReviewedAt = u.ReviewedAt
	}

	return res
}
