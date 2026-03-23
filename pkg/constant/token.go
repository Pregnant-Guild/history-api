package constant

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