package services

import (
	"context"
	"history-api/internal/dtos/response"
	"history-api/internal/models"
	"history-api/internal/repositories"
	"history-api/pkg/convert"

	"github.com/gofiber/fiber/v3"
)

type RoleService interface {
	GetRoleByID(ctx context.Context, id string) (*response.RoleResponse, error)
	GetAllRole(ctx context.Context) ([]*response.RoleResponse, error)
}

type roleService struct {
	roleRepo repositories.RoleRepository
}


func NewRoleService(
	roleRepo repositories.RoleRepository,
) RoleService {
	return &roleService{
		roleRepo: roleRepo,
	}
}

func (r *roleService) GetAllRole(ctx context.Context) ([]*response.RoleResponse, error) {
	roles, err := r.roleRepo.All(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return models.RolesEntityToResponse(roles), nil
}

func (r *roleService) GetRoleByID(ctx context.Context, id string) (*response.RoleResponse, error) {
	roleId, err := convert.StringToUUID(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	role, err := r.roleRepo.GetByID(ctx, roleId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Role not found")
	}

	return role.ToResponse(), nil
}