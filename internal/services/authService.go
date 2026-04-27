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
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Signin(ctx context.Context, dto *request.SignInDto) (*response.AuthResponse, *fiber.Error)
	Signup(ctx context.Context, dto *request.SignUpDto) (*response.AuthResponse, *fiber.Error)
	Logout(ctx context.Context, userId string) *fiber.Error
	ForgotPassword(ctx context.Context, dto *request.ForgotPasswordDto) *fiber.Error
	VerifyToken(ctx context.Context, dto *request.VerifyTokenDto) (*response.VerifyTokenResponse, *fiber.Error)
	CreateToken(ctx context.Context, dto *request.CreateTokenDto) *fiber.Error
	SigninWithGoogle(ctx context.Context, dto *request.SigninWithGoogleDto) (*response.AuthResponse, *fiber.Error)
	RefreshToken(ctx context.Context, id string, refreshToken string) (*response.AuthResponse, *fiber.Error)
}

type authService struct {
	userRepo  repositories.UserRepository
	roleRepo  repositories.RoleRepository
	tokenRepo repositories.TokenRepository
	c         cache.Cache
	db        *pgxpool.Pool
}

func NewAuthService(
	userRepo repositories.UserRepository,
	roleRepo repositories.RoleRepository,
	tokenRepo repositories.TokenRepository,
	c cache.Cache,
	db *pgxpool.Pool,
) AuthService {
	return &authService{
		userRepo:  userRepo,
		roleRepo:  roleRepo,
		tokenRepo: tokenRepo,
		c:         c,
		db:        db,
	}
}

func (a *authService) genToken(user *models.UserEntity) (*response.AuthResponse, error) {
	jwtSecret, err := config.GetConfig("JWT_SECRET")
	if err != nil {
		return nil, err
	}
	jwtRefreshSecret, err := config.GetConfig("JWT_REFRESH_SECRET")
	if err != nil {
		return nil, err
	}

	if jwtSecret == "" || jwtRefreshSecret == "" {
		return nil, errors.New("missing JWT secrets in environment")
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

func (a *authService) Signin(ctx context.Context, dto *request.SignInDto) (*response.AuthResponse, *fiber.Error) {
	if !constants.EMAIL_REGEX.MatchString(dto.Email) {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid email format")
	}

	err := constants.ValidatePassword(dto.Password)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	user, err := a.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil || user == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid email or password")
	}

	if user.AuthProvider != constants.ProviderTypeLocal.String() && user.PasswordHash == "" {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Please sign in with "+user.AuthProvider)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(dto.Password)); err != nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid email or password")
	}

	data, err := a.genToken(user)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to generate security tokens")

	}
	pgID, err := convert.StringToUUID(user.ID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Invalid user ID internal format")
	}
	err = a.userRepo.UpdateRefreshToken(
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
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to update refresh token")
	}

	return data, nil

}

func (a *authService) Logout(ctx context.Context, userId string) *fiber.Error {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer tx.Rollback(ctx)

	uRepoTx := a.userRepo.WithTx(tx)

	pgID, err := convert.StringToUUID(userId)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid user ID format")
	}
	user, err := a.userRepo.GetByID(ctx, pgID)
	if err != nil || user == nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	err = uRepoTx.UpdateTokenVersion(ctx, sqlc.UpdateTokenVersionParams{
		ID:           pgID,
		TokenVersion: user.TokenVersion + 1,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to revoke sessions")
	}

	err = uRepoTx.UpdateRefreshToken(ctx, sqlc.UpdateUserRefreshTokenParams{
		ID: pgID,
		RefreshToken: pgtype.Text{
			String: "",
			Valid:  false,
		},
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to clear refresh token")
	}
	err = tx.Commit(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to commit logout")
	}
	return nil
}

func (a *authService) RefreshToken(ctx context.Context, id string, refreshToken string) (*response.AuthResponse, *fiber.Error) {
	var pgID pgtype.UUID
	err := pgID.Scan(id)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid user ID format")
	}
	user, err := a.userRepo.GetByID(ctx, pgID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	if user.RefreshToken != refreshToken {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid or expired refresh token")
	}

	roles := models.RolesEntityToRoleConstant(user.Roles)

	if slices.Contains(roles, constants.RoleTypeBanned) {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Your account has been banned")
	}

	data, err := a.genToken(user)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to generate new tokens")
	}

	err = a.userRepo.UpdateRefreshToken(
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
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to update refresh token")
	}

	return data, nil
}

func (a *authService) Signup(ctx context.Context, dto *request.SignUpDto) (*response.AuthResponse, *fiber.Error) {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer tx.Rollback(ctx)

	uRepoTx := a.userRepo.WithTx(tx)
	rRepoTx := a.roleRepo.WithTx(tx)

	if !constants.EMAIL_REGEX.MatchString(dto.Email) {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid email format")
	}
	err = constants.ValidatePassword(dto.Password)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ok, err := a.tokenRepo.CheckVerified(ctx, dto.Email, constants.TokenTypeEmailVerify, dto.TokenID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to verify registration token")
	}

	if !ok {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Invalid or expired verification token")
	}

	user, err := a.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to check existing user")
	}
	if user != nil {
		return nil, fiber.NewError(fiber.StatusConflict, "Email is already registered")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to hash password")
	}

	user, err = uRepoTx.UpsertUser(
		ctx,
		sqlc.UpsertUserParams{
			Email: dto.Email,
			PasswordHash: pgtype.Text{
				String: string(hashed),
				Valid:  len(hashed) != 0,
			},
			AuthProvider: constants.ProviderTypeLocal.String(),
		},
	)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create user account")
	}

	userId, err := convert.StringToUUID(user.ID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Invalid user ID internal format")
	}
	_, err = uRepoTx.CreateProfile(
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
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create user profile")
	}
	role, err := a.roleRepo.GetByName(ctx, constants.RoleTypeUser.String())
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Default role not found")
	}

	roleId, err := convert.StringToUUID(role.ID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Invalid role ID internal format")
	}

	err = rRepoTx.CreateUserRole(
		ctx,
		sqlc.CreateUserRoleParams{
			UserID:  userId,
			Column2: []pgtype.UUID{roleId},
		},
	)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to assign user role")
	}

	data, err := a.genToken(user)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to generate security tokens")
	}

	err = uRepoTx.UpdateRefreshToken(
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
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to update refresh token")
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to commit signup")
	}

	return data, nil
}

func (a *authService) ForgotPassword(ctx context.Context, dto *request.ForgotPasswordDto) *fiber.Error {
	ok, err := a.tokenRepo.CheckVerified(ctx, dto.Email, constants.TokenTypePasswordReset, dto.TokenID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to verify reset token")
	}
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid or expired reset token")
	}
	user, err := a.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}
	if user == nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(dto.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to hash new password")
	}
	userId, err := convert.StringToUUID(user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Invalid user ID format")
	}
	err = a.userRepo.UpdatePassword(ctx, sqlc.UpdateUserPasswordParams{
		ID: userId,
		PasswordHash: pgtype.Text{
			String: string(hashed),
			Valid:  len(hashed) != 0,
		},
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update password")
	}
	return nil
}

func (a *authService) SigninWithGoogle(ctx context.Context, dto *request.SigninWithGoogleDto) (*response.AuthResponse, *fiber.Error) {
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer tx.Rollback(ctx)

	uRepoTx := a.userRepo.WithTx(tx)
	rRepoTx := a.roleRepo.WithTx(tx)

	user, err := a.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch user data")
	}

	if user != nil {
		userId, err := convert.StringToUUID(user.ID)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Invalid user ID format")
		}
		data, err := a.genToken(user)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to generate security tokens")
		}
		err = uRepoTx.UpdateRefreshToken(
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
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to update refresh token")
		}
		err = tx.Commit(ctx)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to commit Google signin")
		}
		return data, nil
	}

	user, err = uRepoTx.UpsertUser(
		ctx,
		sqlc.UpsertUserParams{
			Email:        dto.Email,
			AuthProvider: constants.ProviderTypeGoogle.String(),
			GoogleID: pgtype.Text{
				String: dto.Sub,
				Valid:  dto.Sub != "",
			},
		},
	)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create user account")
	}
	userId, err := convert.StringToUUID(user.ID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Invalid user ID format")
	}
	_, err = uRepoTx.CreateProfile(
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
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create user profile")
	}
	role, err := a.roleRepo.GetByName(ctx, constants.RoleTypeUser.String())
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Default role not found")
	}

	roleId, err := convert.StringToUUID(role.ID)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Invalid role ID format")
	}

	err = rRepoTx.CreateUserRole(
		ctx,
		sqlc.CreateUserRoleParams{
			UserID:  userId,
			Column2: []pgtype.UUID{roleId},
		},
	)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to assign user role")
	}

	data, err := a.genToken(user)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to generate security tokens")
	}
	err = uRepoTx.UpdateRefreshToken(
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
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to update refresh token")
	}
	err = tx.Commit(ctx)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to commit Google signin")
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

func (a *authService) CreateToken(ctx context.Context, dto *request.CreateTokenDto) *fiber.Error {
	ok, err := a.tokenRepo.CheckCooldown(ctx, dto.Email, dto.TokenType)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to check request cooldown")
	}

	if ok {
		return fiber.NewError(fiber.StatusTooManyRequests, "Too many requests. Please try again later.")
	}

	user, err := a.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to check user existence")
	}

	shouldSend := true
	if (dto.TokenType == constants.TokenTypeEmailVerify && user != nil) ||
		(dto.TokenType == constants.TokenTypePasswordReset && user == nil) {
		shouldSend = false
	}

	if shouldSend {
		otp, err := a.GenerateOTP()
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to generate OTP")
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
			return fiber.NewError(fiber.StatusInternalServerError, "Failed to save verification token")
		}

		token.Token = otp
		a.c.PublishTask(ctx, constants.StreamEmailName, constants.TaskTypeSendEmailOTP, token)
	}

	return nil
}

func (a *authService) VerifyToken(ctx context.Context, dto *request.VerifyTokenDto) (*response.VerifyTokenResponse, *fiber.Error) {
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
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to check user existence")
	}

	if (dto.TokenType == constants.TokenTypeEmailVerify && user != nil) ||
		(dto.TokenType == constants.TokenTypePasswordReset && user == nil) {
		return nil, genericError
	}

	tokenId := uuid.New().String()
	err = a.tokenRepo.CreateVerified(ctx, dto.Email, dto.TokenType, tokenId)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create verified token record")
	}

	_ = a.tokenRepo.Delete(ctx, dto.Email, dto.TokenType)

	return &response.VerifyTokenResponse{
		TokenID: tokenId,
	}, nil
}
