package repositories

import (
	"context"
	"fmt"
	"history-api/internal/models"
	"history-api/pkg/cache"
	"history-api/pkg/constants"
)

type TokenRepository interface {
	CheckCooldown(ctx context.Context, email string, tokenType constants.TokenType) (bool, error)
	Get(ctx context.Context, email string, tokenType constants.TokenType) (*models.TokenEntity, error)
	Create(ctx context.Context, token *models.TokenEntity) error
	Delete(ctx context.Context, email string, tokenType constants.TokenType) error

	CheckVerified(ctx context.Context, email string, tokenType constants.TokenType, id string) (bool, error)
	CreateVerified(ctx context.Context, email string, tokenType constants.TokenType, id string) error
	DeleteVerified(ctx context.Context, email string, tokenType constants.TokenType, id string) error

	CreateUploadToken(ctx context.Context, userId string, token *models.TokenUploadEntity) error
	GetUploadToken(ctx context.Context, userId string, id string) (*models.TokenUploadEntity, error)
	DeleteUploadToken(ctx context.Context, userId string, id string) error
}

type tokenRepository struct {
	c cache.Cache
}

func NewTokenRepository(c cache.Cache) TokenRepository {
	return &tokenRepository{
		c: c,
	}
}

func (t *tokenRepository) CreateVerified(ctx context.Context, email string, tokenType constants.TokenType, id string) error {
	cacheKey := fmt.Sprintf("token:verified:%d:%s:%s", tokenType.Value(), email, id)
	return t.c.Set(ctx, cacheKey, true, constants.TokenVerifiedDuration)
}

func (t *tokenRepository) DeleteVerified(ctx context.Context, email string, tokenType constants.TokenType, id string) error {
	cacheKey := fmt.Sprintf("token:verified:%d:%s:%s", tokenType.Value(), email, id)
	return t.c.Del(ctx, cacheKey)
}
func (t *tokenRepository) CheckVerified(ctx context.Context, email string, tokenType constants.TokenType, id string) (bool, error) {
	cacheKey := fmt.Sprintf("token:verified:%d:%s:%s", tokenType.Value(), email, id)
	exists, err := t.c.Exists(ctx, cacheKey)
	return exists, err
}

func (t *tokenRepository) CreateUploadToken(ctx context.Context, userId string, token *models.TokenUploadEntity) error {
	cacheKey := fmt.Sprintf("token:%d:%s:%s", constants.TokenUpload.Value(), userId, token.ID)
	err := t.c.Set(ctx, cacheKey, token, constants.TokenUploadDuration)
	if err != nil {
		return err
	}
	return nil
}

func (t *tokenRepository) GetUploadToken(ctx context.Context, userId string, id string) (*models.TokenUploadEntity, error) {
	cacheKey := fmt.Sprintf("token:%d:%s:%s", constants.TokenUpload.Value(), userId, id)
	var token models.TokenUploadEntity
	err := t.c.Get(ctx, cacheKey, &token)
	if err != nil {
		return nil, err
	}
	return &token, err
}

func (t *tokenRepository) DeleteUploadToken(ctx context.Context, userId string, id string) error {
	cacheKey := fmt.Sprintf("token:%d:%s:%s", constants.TokenUpload.Value(), userId, id)
	return t.c.Del(ctx, cacheKey)
}

func (t *tokenRepository) CheckCooldown(ctx context.Context, email string, tokenType constants.TokenType) (bool, error) {
	cacheKey := fmt.Sprintf("token:cooldown:%d:%s", tokenType.Value(), email)
	exists, err := t.c.Exists(ctx, cacheKey)
	return exists, err
}

func (t *tokenRepository) Create(ctx context.Context, token *models.TokenEntity) error {
	cacheKey := fmt.Sprintf("token:%d:%s", token.TokenType.Value(), token.Email)
	err := t.c.Set(ctx, cacheKey, token, constants.TokenExpirationDuration)
	if err != nil {
		return err
	}
	cooldownKey := fmt.Sprintf("token:cooldown:%d:%s", token.TokenType.Value(), token.Email)
	return t.c.Set(ctx, cooldownKey, true, constants.TokenCooldownDuration)
}

func (t *tokenRepository) Delete(ctx context.Context, email string, tokenType constants.TokenType) error {
	cacheKey := fmt.Sprintf("token:%d:%s", tokenType.Value(), email)
	cooldownKey := fmt.Sprintf("token:cooldown:%d:%s", tokenType.Value(), email)
	_ = t.c.Del(ctx, cooldownKey)
	return t.c.Del(ctx, cacheKey)
}

func (t *tokenRepository) Get(ctx context.Context, email string, tokenType constants.TokenType) (*models.TokenEntity, error) {
	cacheKey := fmt.Sprintf("token:%d:%s", tokenType.Value(), email)
	var token models.TokenEntity
	err := t.c.Get(ctx, cacheKey, &token)
	if err != nil {
		return nil, err
	}
	return &token, nil
}
