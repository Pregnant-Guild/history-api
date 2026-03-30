package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
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
	SigninWithGoogle(ctx context.Context, dto *request.SigninWithGoogleDto) (*response.AuthResponse, error)
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

	if user.AuthProvider != constants.LocalProvider.String() && user.PasswordHash == "" {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Please sign in with "+user.AuthProvider)
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
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
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

func (a *authService) SigninWithGoogle(ctx context.Context, dto *request.SigninWithGoogleDto) (*response.AuthResponse, error) {
	user, err := a.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if user != nil {
		userId, err := convert.StringToUUID(user.ID)
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

	user, err = a.userRepo.UpsertUser(
		ctx,
		sqlc.UpsertUserParams{
			Email:        dto.Email,
			AuthProvider: constants.GoogleProvider.String(),
			GoogleID: pgtype.Text{
				String: dto.Sub,
				Valid:  dto.Sub != "",
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
				String: dto.Name,
				Valid:  dto.Name != "",
			},
			AvatarUrl: pgtype.Text{
				String: dto.Picture,
				Valid:  dto.Picture != "",
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
		return fiber.NewError(fiber.StatusInternalServerError, "Internal Server Error")
	}

	if ok {
		return fiber.NewError(fiber.StatusBadRequest, "Too many requests. Please try again later.")
	}

	user, err := a.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fiber.NewError(fiber.StatusInternalServerError, "Internal Server Error")
	}

	shouldSend := true
	if (dto.TokenType == constants.TokenEmailVerify && user != nil) ||
		(dto.TokenType == constants.TokenPasswordReset && user == nil) {
		shouldSend = false
	}

	if shouldSend {
		otp, err := a.GenerateOTP()
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal Server Error")
		}
		hash := sha256.Sum256([]byte(otp))
		hashString := hex.EncodeToString(hash[:])
		token := &models.TokenEntity{
			Email:     dto.Email,
			Token:     hashString,
			TokenType: dto.TokenType,
		}
		err = a.tokenRepo.Create(ctx, token)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Internal Server Error")
		}

		token.Token = otp
		a.c.PublishTask(ctx, constants.StreamEmailName, constants.TaskTypeSendEmailOTP, token)
	}

	return nil
}

func (a *authService) VerifyToken(ctx context.Context, dto *request.VerifyTokenDto) (*response.VerifyTokenResponse, error) {
	genericError := fiber.NewError(fiber.StatusBadRequest, "Invalid or expired token")
	token, err := a.tokenRepo.Get(ctx, dto.Email, dto.TokenType)
	if err != nil || token == nil {
		return nil, genericError
	}

	userOtpHash := sha256.Sum256([]byte(dto.Token))
	userOtpHashString := hex.EncodeToString(userOtpHash[:])
	actualHash := []byte(token.Token)
	expectedHash := []byte(userOtpHashString)

	if len(actualHash) != len(expectedHash) {
		return nil, genericError
	}

	if subtle.ConstantTimeCompare(actualHash, expectedHash) != 1 {
		return nil, genericError
	}

	user, err := a.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Internal Server Error")
	}

	if (dto.TokenType == constants.TokenEmailVerify && user != nil) ||
		(dto.TokenType == constants.TokenPasswordReset && user == nil) {
		return nil, genericError
	}

	tokenId := uuid.New().String()
	err = a.tokenRepo.CreateVerified(ctx, dto.Email, dto.TokenType, tokenId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Internal Server Error")
	}

	_ = a.tokenRepo.Delete(ctx, dto.Email, dto.TokenType)

	return &response.VerifyTokenResponse{
		TokenID: tokenId,
	}, nil
}
