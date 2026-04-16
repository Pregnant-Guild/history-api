package models

import (
	"encoding/json"
	"history-api/internal/dtos/response"
	"history-api/pkg/constants"
	"time"
)

type UserVerificationEntity struct {
	ID         string               `json:"id"`
	User       *UserSimpleEntity    `json:"user"`
	VerifyType constants.VerifyType `json:"verify_type"`
	Content    string               `json:"content"`
	IsDeleted  bool                 `json:"is_deleted"`
	Status     constants.StatusType `json:"status"`
	Reviewer   *UserSimpleEntity    `json:"reviewer"`
	ReviewNote string               `json:"review_note"`
	ReviewedAt *time.Time           `json:"reviewed_at"`
	CreatedAt  *time.Time           `json:"created_at"`
	Media      []*MediaSimpleEntity `json:"media"`
}

type UserVerificationStorageEntity struct {
	Email      string               `json:"email"`
	Name       string               `json:"name"`
	Status     constants.StatusType `json:"status"`
	ReviewNote string               `json:"review_note"`
}

func (u *UserVerificationEntity) ParseMedia(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		u.Media = []*MediaSimpleEntity{}
		return nil
	}
	return json.Unmarshal(data, &u.Media)
}

func (u *UserVerificationEntity) ParseUser(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		u.User = nil
		return nil
	}
	return json.Unmarshal(data, &u.User)
}

func (u *UserVerificationEntity) ParseReviewer(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		u.Reviewer = nil
		return nil
	}
	return json.Unmarshal(data, &u.Reviewer)
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
		User:       u.User.ToResponse(),
		VerifyType: u.VerifyType.String(),
		Content:    u.Content,
		Status:     u.Status.String(),
		ReviewNote: u.ReviewNote,
		Reviewer:   u.Reviewer.ToResponse(),
		ReviewedAt: u.ReviewedAt,
		CreatedAt:  u.CreatedAt,
		Medias:     mediaResponses,
	}

	if u.ReviewedAt != nil {
		res.ReviewedAt = u.ReviewedAt
	}

	return res
}

func UserVerificationsEntitiesToResponse(entities []*UserVerificationEntity) []*response.UserVerificationResponse {
	responses := make([]*response.UserVerificationResponse, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		responses = append(responses, entity.ToResponse())

	}
	return responses
}
