package models

import (
	"history-api/internal/dtos/response"
	"history-api/pkg/constants"
	"time"
)

type RoleSimple struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (r *RoleSimple) ToResponse() *response.RoleSimpleResponse {
	if r == nil {
		return nil
	}
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
	if r == nil {
		return nil
	}
	return &response.RoleResponse{
		ID:        r.ID,
		Name:      r.Name,
		IsDeleted: r.IsDeleted,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func (r *RoleEntity) ToRoleSimple() *RoleSimple {
	if r == nil {
		return nil
	}
	return &RoleSimple{
		ID:   r.ID,
		Name: r.Name,
	}
}

func RolesEntityToResponse(rs []*RoleEntity) []*response.RoleResponse {
	out := make([]*response.RoleResponse, len(rs))
	for _, role := range rs {
		if role == nil {
			continue
		}
		out = append(out, role.ToResponse())
	}
	return out
}

func RolesEntityToRoleConstant(rs []*RoleSimple) []constants.Role {
	out := make([]constants.Role, len(rs))
	for _, role := range rs {
		data, ok := constants.ParseRole(role.Name)
		if !ok {
			continue
		}
		out= append(out, data)
	}
	return out
}
