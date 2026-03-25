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
	IsVerified   bool               `json:"is_verified"`
	TokenVersion int32              `json:"token_version"`
	GoogleID     string             `json:"google_id"`
	AuthProvider string             `json:"auth_provider"`
	RefreshToken string             `json:"refresh_token"`
	IsDeleted    bool               `json:"is_deleted"`
	CreatedAt    *time.Time         `json:"created_at"`
	UpdatedAt    *time.Time         `json:"updated_at"`
	Roles        []*RoleSimple      `json:"roles"`
}

func (u *UserEntity) ParseRoles(data []byte) error {
	if len(data) == 0 {
		u.Roles = []*RoleSimple{}
		return nil
	}
	return json.Unmarshal(data, &u.Roles)
}

func (u *UserEntity) ParseProfile(data []byte) error {
	if len(data) == 0 {
		u.Profile = &UserProfileSimple{}
		return nil
	}
	return json.Unmarshal(data, &u.Profile)
}

func (u *UserEntity) ToResponse() *response.UserResponse {
	return &response.UserResponse{
		ID:           u.ID,
		Email:        u.Email,
		IsVerified:   u.IsVerified,
		TokenVersion: u.TokenVersion,
		IsDeleted:    u.IsDeleted,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
		Roles:        RolesToResponse(u.Roles),
		Profile:      u.Profile.ToResponse(),
	}
}

func UsersEntityToResponse(rs []*UserEntity) []*response.UserResponse {
	out := make([]*response.UserResponse, len(rs))
	for i := range rs {
		out[i] = rs[i].ToResponse()
	}
	return out
}
