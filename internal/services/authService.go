package services

import (
	"context"
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/internal/repositories"
	"history-api/pkg/config"
	"history-api/pkg/constant"

	"slices"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Signin(ctx context.Context, dto *request.SignInDto) (*response.AuthResponse, error)
	Signup(ctx context.Context, dto *request.SignUpDto) (*response.AuthResponse, error)
	ForgotPassword(ctx context.Context) error
	VerifyToken(ctx context.Context) error
	CreateToken(ctx context.Context) error
	SigninWith3rd(ctx context.Context) error
	RefreshToken(ctx context.Context, id string) (*response.AuthResponse, error)
}

type authService struct {
	userRepo repositories.UserRepository
	roleRepo repositories.RoleRepository
}

func NewAuthService(
	userRepo repositories.UserRepository,
	roleRepo repositories.RoleRepository,
) AuthService {
	return &authService{
		userRepo: userRepo,
		roleRepo: roleRepo,
	}
}

func (a *authService) genToken(Uid string, role []constant.Role) (*response.AuthResponse, error) {
	jwtSecret, err := config.GetConfig("JWT_SECRET")
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "missing JWT_SECRET in environment")
	}
	jwtRefreshSecret, err := config.GetConfig("JWT_REFRESH_SECRET")
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "missing JWT_REFRESH_SECRET in environment")
	}

	if jwtSecret == "" || jwtRefreshSecret == "" {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "missing JWT secrets in environment")
	}

	claimsAccess := &response.JWTClaims{
		UId:   Uid,
		Roles: role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Minute)),
		},
	}

	claimsRefresh := &response.JWTClaims{
		UId:   Uid,
		Roles: role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * 24 * time.Hour)),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsAccess)
	at, err := accessToken.SignedString([]byte(jwtSecret))
	if err != nil {
		return nil, err
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsRefresh)
	rt, err := refreshToken.SignedString([]byte(jwtRefreshSecret))
	if err != nil {
		return nil, err
	}

	res := response.AuthResponse{
		AccessToken:  at,
		RefreshToken: rt,
	}
	return &res, nil
}

func (a *authService) saveNewRefreshToken(ctx context.Context, params sqlc.UpdateUserRefreshTokenParams) error {
	err := a.userRepo.UpdateRefreshToken(ctx, params)
	if err != nil {
		return err
	}
	return nil
}

func (a *authService) Signin(ctx context.Context, dto *request.SignInDto) (*response.AuthResponse, error) {
	if !constant.EMAIL_REGEX.MatchString(dto.Email) {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid email")
	}

	err := constant.ValidatePassword(dto.Password)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	user, err := a.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil || user == nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(dto.Password)); err != nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid identity or password!")
	}

	data, err := a.genToken(user.ID, models.RolesEntityToRoleConstant(user.Roles))
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())

	}
	var pgID pgtype.UUID
	err = pgID.Scan(user.ID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	err = a.saveNewRefreshToken(
		ctx,
		sqlc.UpdateUserRefreshTokenParams{
			ID: pgID,
			RefreshToken: pgtype.Text{
				String: data.RefreshToken,
				Valid:  data.RefreshToken != "",
			},
		},
	)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return data, nil

}

func (a *authService) RefreshToken(ctx context.Context, id string) (*response.AuthResponse, error) {
	var pgID pgtype.UUID
	err := pgID.Scan(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	user, err := a.userRepo.GetByID(ctx, pgID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Invalid user data")
	}
	roles := models.RolesEntityToRoleConstant(user.Roles)

	if slices.Contains(roles, constant.BANNED) {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "User is banned!")
	}

	data, err := a.genToken(id, roles)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	err = a.saveNewRefreshToken(
		ctx,
		sqlc.UpdateUserRefreshTokenParams{
			ID: pgID,
			RefreshToken: pgtype.Text{
				String: data.RefreshToken,
				Valid:  data.RefreshToken != "",
			},
		},
	)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return data, nil
}

func (a *authService) Signup(ctx context.Context, dto *request.SignUpDto) (*response.AuthResponse, error) {
	if !constant.EMAIL_REGEX.MatchString(dto.Email) {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid email")
	}
	err := constant.ValidatePassword(dto.Password)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	user, err := a.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if user != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "User already exists")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	user, err = a.userRepo.UpsertUser(
		ctx,
		sqlc.UpsertUserParams{
			Email: dto.Email,
			PasswordHash: pgtype.Text{
				String: string(hashed),
				Valid:  len(hashed) != 0,
			},
			IsVerified: true,
		},
	)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	var userId pgtype.UUID
	err = userId.Scan(user.ID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	_, err = a.userRepo.CreateProfile(
		ctx,
		sqlc.CreateUserProfileParams{
			UserID: userId,
			DisplayName: pgtype.Text{
				String: dto.DisplayName,
				Valid:  dto.DisplayName != "",
			},
		},
	)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	err = a.roleRepo.AddUserRole(
		ctx,
		sqlc.AddUserRoleParams{
			UserID: userId,
			Name:   constant.USER.String(),
		},
	)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	data, err := a.genToken(user.ID, constant.USER.ToSlice())
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	err = a.saveNewRefreshToken(
		ctx,
		sqlc.UpdateUserRefreshTokenParams{
			ID: userId,
			RefreshToken: pgtype.Text{
				String: data.RefreshToken,
				Valid:  data.RefreshToken != "",
			},
		},
	)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return data, nil
}

// ForgotPassword implements [AuthService].
func (a *authService) ForgotPassword(ctx context.Context) error {
	panic("unimplemented")
}

// SigninWith3rd implements [AuthService].
func (a *authService) SigninWith3rd(ctx context.Context) error {
	panic("unimplemented")
}

// CreateToken implements [AuthService].
func (a *authService) CreateToken(ctx context.Context) error {
	panic("unimplemented")
}

// Verify implements [AuthService].
func (a *authService) VerifyToken(ctx context.Context) error {
	panic("unimplemented")
}
