package models

import (
	"history-api/internal/dtos/response"
	"history-api/pkg/constant"
	"time"
)

type RoleSimple struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (r *RoleSimple) ToResponse() *response.RoleSimpleResponse {
	return &response.RoleSimpleResponse{
		ID:   r.ID,
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
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	IsDeleted bool       `json:"is_deleted"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

func (r *RoleEntity) ToResponse() *response.RoleResponse {
	return &response.RoleResponse{
		ID:        r.ID,
		Name:      r.Name,
		IsDeleted: r.IsDeleted,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func RolesEntityToResponse(rs []*RoleEntity) []*response.RoleResponse {
	out := make([]*response.RoleResponse, len(rs))
	for i := range rs {
		out[i] = rs[i].ToResponse()
	}
	return out
}

func RolesEntityToRoleConstant(rs []*RoleSimple) []constant.Role {
	out := make([]constant.Role, len(rs))
	for i := range rs {
		data, ok := constant.ParseRole(rs[i].Name)
		if !ok {
			continue
		}
		out[i] = data
	}
	return out
}
