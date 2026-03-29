package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"history-api/internal/dtos/request"
	"history-api/internal/dtos/response"
	"history-api/internal/gen/sqlc"
	"history-api/internal/models"
	"history-api/internal/repositories"
	"history-api/pkg/cache"
	"history-api/pkg/config"
	"history-api/pkg/constants"
	"history-api/pkg/convert"
	"math/big"

	"slices"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Signin(ctx context.Context, dto *request.SignInDto) (*response.AuthResponse, error)
	Signup(ctx context.Context, dto *request.SignUpDto) (*response.AuthResponse, error)
	ForgotPassword(ctx context.Context, dto *request.ForgotPasswordDto) error
	VerifyToken(ctx context.Context, dto *request.VerifyTokenDto) (*response.VerifyTokenResponse, error)
	CreateToken(ctx context.Context, dto *request.CreateTokenDto) error
	SigninWith3rd(ctx context.Context, dto *request.SigninWith3rdDto) error
	RefreshToken(ctx context.Context, id string) (*response.AuthResponse, error)
}

type authService struct {
	userRepo  repositories.UserRepository
	roleRepo  repositories.RoleRepository
	tokenRepo repositories.TokenRepository
	c         cache.Cache
}

func NewAuthService(
	userRepo repositories.UserRepository,
	roleRepo repositories.RoleRepository,
	tokenRepo repositories.TokenRepository,
	c cache.Cache,
) AuthService {
	return &authService{
		userRepo:  userRepo,
		roleRepo:  roleRepo,
		tokenRepo: tokenRepo,
		c:         c,
	}
}

func (a *authService) genToken(user *models.UserEntity) (*response.AuthResponse, error) {
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
		UId:          user.ID,
		Roles:        models.RolesEntityToRoleConstant(user.Roles),
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(constants.AccessTokenDuration)),
		},
	}

	claimsRefresh := &response.JWTClaims{
		UId:          user.ID,
		Roles:        models.RolesEntityToRoleConstant(user.Roles),
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(constants.RefreshTokenDuration)),
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
	if !constants.EMAIL_REGEX.MatchString(dto.Email) {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid email")
	}

	err := constants.ValidatePassword(dto.Password)
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

	data, err := a.genToken(user)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())

	}
	pgID, err := convert.StringToUUID(user.ID)
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

	if slices.Contains(roles, constants.BANNED) {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "User is banned!")
	}

	data, err := a.genToken(user)
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
	if !constants.EMAIL_REGEX.MatchString(dto.Email) {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid email")
	}
	err := constants.ValidatePassword(dto.Password)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ok, err := a.tokenRepo.CheckVerified(ctx, dto.Email, constants.TokenEmailVerify, dto.TokenID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if !ok {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid or expired token")
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
		},
	)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	userId, err := convert.StringToUUID(user.ID)
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
	role, err := a.roleRepo.GetByname(ctx, constants.USER.String())
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	roleId, err := convert.StringToUUID(role.ID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	err = a.roleRepo.AddUserRole(
		ctx,
		sqlc.AddUserRoleParams{
			UserID:  userId,
			Column2: []pgtype.UUID{roleId},
		},
	)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	data, err := a.genToken(user)
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

func (a *authService) ForgotPassword(ctx context.Context, dto *request.ForgotPasswordDto) error {
	ok, err := a.tokenRepo.CheckVerified(ctx, dto.Email, constants.TokenPasswordReset, dto.TokenID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid or expired token")
	}
	user, err := a.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if user == nil {
		return fiber.NewError(fiber.StatusBadRequest, "User not found")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(dto.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	userId, err := convert.StringToUUID(user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	err = a.userRepo.UpdatePassword(ctx, sqlc.UpdateUserPasswordParams{
		ID: userId,
		PasswordHash: pgtype.Text{
			String: string(hashed),
			Valid:  len(hashed) != 0,
		},
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return nil
}

// SigninWith3rd implements [AuthService].
func (a *authService) SigninWith3rd(ctx context.Context, dto *request.SigninWith3rdDto) error {
	panic("unimplemented")
}
func (a *authService) GenerateOTP() (string, error) {
	max := big.NewInt(900000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	otp := n.Int64() + 100000
	return fmt.Sprintf("%06d", otp), nil
}

func (a *authService) CreateToken(ctx context.Context, dto *request.CreateTokenDto) error {
	ok, err := a.tokenRepo.CheckCooldown(ctx, dto.Email, dto.TokenType)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if ok {
		return fiber.NewError(fiber.StatusBadRequest, "Please wait before requesting another token")
	}

	otp, err := a.GenerateOTP()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	token := &models.TokenEntity{
		Email:     dto.Email,
		Token:     otp,
		TokenType: dto.TokenType,
	}

	err = a.tokenRepo.Create(ctx, token)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	a.c.PublishTask(ctx, constants.StreamEmailName, constants.TaskTypeSendEmailOTP, token)
	return nil
}

func (a *authService) VerifyToken(ctx context.Context, dto *request.VerifyTokenDto) (*response.VerifyTokenResponse, error) {
	token, err := a.tokenRepo.Get(ctx, dto.Email, dto.TokenType)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if token == nil || token.Token != dto.Token {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid token")
	}
	tokenId := uuid.New().String()
	err = a.tokenRepo.CreateVerified(ctx, dto.Email, dto.TokenType, tokenId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return &response.VerifyTokenResponse{
		TokenID: tokenId,
	}, nil
}
