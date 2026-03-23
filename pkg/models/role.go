package models

import (
	"history-api/pkg/dtos/response"
	"history-api/pkg/convert"

	"github.com/jackc/pgx/v5/pgtype"
)

type RoleSimple struct {
	ID   pgtype.UUID `json:"id"`
	Name string      `json:"name"`
}

func (r *RoleSimple) ToResponse() *response.RoleSimpleResponse {
	return &response.RoleSimpleResponse{
		ID:   convert.UUIDToString(r.ID),
		Name: r.Name,
	}
}

func RolesToResponse(rs []*RoleSimple) []*response.RoleSimpleResponse {
	out := make([]*response.RoleSimpleResponse, len(rs))
	for i := range rs {
		out[i] = rs[i].ToResponse()
	}
	return out
}

type RoleEntity struct {
	ID        pgtype.UUID        `json:"id"`
	Name      string             `json:"name"`
	IsDeleted pgtype.Bool        `json:"is_deleted"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
	UpdatedAt pgtype.Timestamptz `json:"updated_at"`
}

func (r *RoleEntity) ToResponse() *response.RoleResponse {
	return &response.RoleResponse{
		ID:        convert.UUIDToString(r.ID),
		Name:      r.Name,
		IsDeleted: convert.BoolVal(r.IsDeleted),
		CreatedAt: convert.TimeToPtr(r.CreatedAt),
		UpdatedAt: convert.TimeToPtr(r.UpdatedAt),
	}
}

func RolesEntityToResponse(rs []*RoleEntity) []*response.RoleResponse {
	out := make([]*response.RoleResponse, len(rs))
	for i := range rs {
		out[i] = rs[i].ToResponse()
	}
	return out
}
