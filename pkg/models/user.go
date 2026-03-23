package models

import (
	"encoding/json"
	"history-api/pkg/convert"
	"history-api/pkg/dtos/response"

	"github.com/jackc/pgx/v5/pgtype"
)

type UserEntity struct {
	ID           pgtype.UUID        `json:"id"`
	Name         string             `json:"name"`
	Email        string             `json:"email"`
	PasswordHash string             `json:"password_hash"`
	AvatarUrl    pgtype.Text        `json:"avatar_url"`
	IsActive     pgtype.Bool        `json:"is_active"`
	IsVerified   pgtype.Bool        `json:"is_verified"`
	TokenVersion int32              `json:"token_version"`
	RefreshToken pgtype.Text        `json:"refresh_token"`
	IsDeleted    pgtype.Bool        `json:"is_deleted"`
	CreatedAt    pgtype.Timestamptz `json:"created_at"`
	UpdatedAt    pgtype.Timestamptz `json:"updated_at"`
	Roles        []*RoleSimple      `json:"roles"`
}

func (u *UserEntity) ParseRoles(data []byte) error {
	if len(data) == 0 {
		u.Roles = []*RoleSimple{}
		return nil
	}
	return json.Unmarshal(data, &u.Roles)
}

func (u *UserEntity) ToResponse() *response.UserResponse {
	return &response.UserResponse{
		ID:           convert.UUIDToString(u.ID),
		Name:         u.Name,
		Email:        u.Email,
		AvatarUrl:    convert.TextToString(u.AvatarUrl),
		IsActive:     convert.BoolVal(u.IsActive),
		IsVerified:   convert.BoolVal(u.IsVerified),
		TokenVersion: u.TokenVersion,
		IsDeleted:    convert.BoolVal(u.IsDeleted),
		CreatedAt:    convert.TimeToPtr(u.CreatedAt),
		UpdatedAt:    convert.TimeToPtr(u.UpdatedAt),
		Roles:        RolesToResponse(u.Roles),
	}
}


func UsersEntityToResponse(rs []*UserEntity) []*response.UserResponse {
	out := make([]*response.UserResponse, len(rs))
	for i := range rs {
		out[i] = rs[i].ToResponse()
	}
	return out
}