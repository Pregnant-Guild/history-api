package services

import (
	"context"
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/repositories"
)

type UserService interface {
	//user
	GetUserCurrent(ctx context.Context, dto *request.SignInDto) (*response.AuthResponse, error)
	UpdateProfile(ctx context.Context, id string) (*response.UserResponse, error)
	ChangePassword(ctx context.Context, id string) (*response.UserResponse, error)

	//admin
	DeleteUser(ctx context.Context, id string) (*response.UserResponse, error)
	ChangeRoleUser(ctx context.Context, id string) (*response.UserResponse, error)
	RestoreUser(ctx context.Context, id string) (*response.UserResponse, error)
	GetUserByID(ctx context.Context, id string) (*response.UserResponse, error)
	Search(ctx context.Context, id string) ([]*response.UserResponse, error)
	GetAllUser(ctx context.Context, id string) ([]*response.UserResponse, error)
}

type userService struct {
	userRepo repositories.UserRepository
	roleRepo repositories.RoleRepository
}

func NewUserService(
	userRepo repositories.UserRepository,
	roleRepo repositories.RoleRepository,
) UserService {
	return &userService{
		userRepo: userRepo,
		roleRepo: roleRepo,
	}
}

// ChangePassword implements [UserService].
func (u *userService) ChangePassword(ctx context.Context, id string) (*response.UserResponse, error) {
	panic("unimplemented")
}

// ChangeRoleUser implements [UserService].
func (u *userService) ChangeRoleUser(ctx context.Context, id string) (*response.UserResponse, error) {
	panic("unimplemented")
}

// DeleteUser implements [UserService].
func (u *userService) DeleteUser(ctx context.Context, id string) (*response.UserResponse, error) {
	panic("unimplemented")
}

// GetAllUser implements [UserService].
func (u *userService) GetAllUser(ctx context.Context, id string) ([]*response.UserResponse, error) {
	panic("unimplemented")
}

// GetUserByID implements [UserService].
func (u *userService) GetUserByID(ctx context.Context, id string) (*response.UserResponse, error) {
	panic("unimplemented")
}

// GetUserCurrent implements [UserService].
func (u *userService) GetUserCurrent(ctx context.Context, dto *request.SignInDto) (*response.AuthResponse, error) {
	panic("unimplemented")
}

// RestoreUser implements [UserService].
func (u *userService) RestoreUser(ctx context.Context, id string) (*response.UserResponse, error) {
	panic("unimplemented")
}

// Search implements [UserService].
func (u *userService) Search(ctx context.Context, id string) ([]*response.UserResponse, error) {
	panic("unimplemented")
}

// UpdateProfile implements [UserService].
func (u *userService) UpdateProfile(ctx context.Context, id string) (*response.UserResponse, error) {
	panic("unimplemented")
}
