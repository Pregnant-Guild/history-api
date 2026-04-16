package models

import (
	"encoding/json"
	"history-api/internal/dtos/response"
	"time"
)

type UserEntity struct {
	ID           string             `json:"id"`
	Email        string             `json:"email"`
	PasswordHash string             `json:"password_hash"`
	Profile      *UserProfileSimple `json:"profile"`
	TokenVersion int32              `json:"token_version"`
	GoogleID     string             `json:"google_id"`
	AuthProvider string             `json:"auth_provider"`
	RefreshToken string             `json:"refresh_token"`
	IsDeleted    bool               `json:"is_deleted"`
	CreatedAt    *time.Time         `json:"created_at"`
	UpdatedAt    *time.Time         `json:"updated_at"`
	Roles        []*RoleSimple      `json:"roles"`
}
type UserSimpleEntity struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	FullName    string `json:"full_name"`
	AvatarUrl   string `json:"avatar_url"`
}

func (u *UserSimpleEntity) ToResponse() *response.UserSimpleResponse {
	if u == nil {
		return nil
	}
	return &response.UserSimpleResponse{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		FullName:    u.FullName,
		AvatarUrl:   u.AvatarUrl,
	}
}

func (u *UserEntity) ParseRoles(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		u.Roles = []*RoleSimple{}
		return nil
	}
	return json.Unmarshal(data, &u.Roles)
}

func (u *UserEntity) ParseProfile(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		u.Profile = &UserProfileSimple{}
		return nil
	}
	return json.Unmarshal(data, &u.Profile)
}

func (u *UserEntity) ToResponse() *response.UserResponse {
	if u == nil {
		return nil
	}
	return &response.UserResponse{
		ID:           u.ID,
		Email:        u.Email,
		TokenVersion: u.TokenVersion,
		IsDeleted:    u.IsDeleted,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
		Roles:        RolesToResponse(u.Roles),
		Profile:      u.Profile.ToResponse(),
	}
}

func UsersEntityToResponse(users []*UserEntity) []*response.UserResponse {
	out := make([]*response.UserResponse, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		out = append(out, user.ToResponse())
	}
	return out
}
