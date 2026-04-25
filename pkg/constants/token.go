package constants

type TokenType int16

const (
	TokenTypePasswordReset TokenType = 1
	TokenTypeEmailVerify   TokenType = 2
	TokenTypeMagicLink     TokenType = 3
	TokenTypeUpload        TokenType = 4
)

func (t TokenType) String() string {
	switch t {
	case TokenTypePasswordReset:
		return "PASSWORD_RESET"
	case TokenTypeEmailVerify:
		return "EMAIL_VERIFY"
	case TokenTypeMagicLink:
		return "LOGIN_MAGIC_LINK"
	case TokenTypeUpload:
		return "UPLOAD"
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
		return TokenTypePasswordReset
	case 2:
		return TokenTypeEmailVerify
	case 3:
		return TokenTypeMagicLink
	case 4:
		return TokenTypeUpload
	default:
		return 0
	}
}

func ParseTokenTypeFromString(s string) TokenType {
	switch s {
	case "PASSWORD_RESET":
		return TokenTypePasswordReset
	case "EMAIL_VERIFY":
		return TokenTypeEmailVerify
	case "LOGIN_MAGIC_LINK":
		return TokenTypeMagicLink
	case "UPLOAD":
		return TokenTypeUpload
	default:
		return 0
	}
}
