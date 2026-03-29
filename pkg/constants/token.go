package constants

type TokenType int16

const (
	TokenPasswordReset TokenType = 1
	TokenEmailVerify   TokenType = 2
	TokenMagicLink     TokenType = 3
	TokenRefreshToken  TokenType = 4
)

func (t TokenType) String() string {
	switch t {
	case TokenPasswordReset:
		return "PASSWORD_RESET"
	case TokenEmailVerify:
		return "EMAIL_VERIFY"
	case TokenMagicLink:
		return "LOGIN_MAGIC_LINK"
	case TokenRefreshToken:
		return "REFRESH_TOKEN"
	default:
		return "UNKNOWN"
	}
}

func (t TokenType) Value() int16 {
	return int16(t)
}

func ParseTokenType(v int16) TokenType {
	switch v {
	case 1:
		return TokenPasswordReset
	case 2:
		return TokenEmailVerify
	case 3:
		return TokenMagicLink
	case 4:
		return TokenRefreshToken
	default:
		return 0
	}
}

func ParseTokenTypeFromString(s string) TokenType {
	switch s {
	case "PASSWORD_RESET":
		return TokenPasswordReset
	case "EMAIL_VERIFY":
		return TokenEmailVerify
	case "LOGIN_MAGIC_LINK":
		return TokenMagicLink
	case "REFRESH_TOKEN":
		return TokenRefreshToken
	default:
		return 0
	}
}