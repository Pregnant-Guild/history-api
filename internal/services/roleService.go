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
	GetRoleByID(ctx context.Context, id string) (*response.RoleResponse, *fiber.Error)
	GetAllRole(ctx context.Context) ([]*response.RoleResponse, *fiber.Error)
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

func (r *roleService) GetAllRole(ctx context.Context) ([]*response.RoleResponse, *fiber.Error) {
	roles, err := r.roleRepo.All(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch roles")
	}

	return models.RolesEntityToResponse(roles), nil
}

func (r *roleService) GetRoleByID(ctx context.Context, id string) (*response.RoleResponse, *fiber.Error) {
	roleId, err := convert.StringToUUID(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid role ID format")
	}
	role, err := r.roleRepo.GetByID(ctx, roleId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Role not found")
	}

	return role.ToResponse(), nil
}
