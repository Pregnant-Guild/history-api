package constants

import "time"

const (
	TokenCooldownDuration   = 1 * time.Minute
	TokenExpirationDuration = 20 * time.Minute
	NormalCacheDuration     = 15 * time.Minute
	ListCacheDuration       = 10 * time.Minute
	AccessTokenDuration     = 15 * time.Minute
	RefreshTokenDuration    = 7 * 24 * time.Hour
	TokenVerifiedDuration   = 10 * time.Minute
)
