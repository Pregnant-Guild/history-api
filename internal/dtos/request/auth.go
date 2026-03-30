package request

import "history-api/pkg/constants"

type SignUpDto struct {
	Email       string `json:"email" validate:"required,min=5,max=255,email"`
	Password    string `json:"password" validate:"required,min=8,max=64"`
	DisplayName string `json:"display_name" validate:"required,min=2,max=50"`
	TokenID     string `json:"token_id" validate:"required,uuid"`
}
type SignInDto struct {
	Email    string `json:"email" validate:"required,min=5,max=255,email"`
	Password string `json:"password" validate:"required,min=8,max=64"`
}

type CreateTokenDto struct {
	Email     string              `json:"email" validate:"required,email"`
	TokenType constants.TokenType `json:"token_type" validate:"required,oneof=1 2 3 4"`
}

type VerifyTokenDto struct {
	Email     string              `json:"email" validate:"required,email"`
	TokenType constants.TokenType `json:"token_type" validate:"required,oneof=1 2 3 4"`
	Token     string              `json:"token" validate:"required,len=6,numeric"`
}

type ForgotPasswordDto struct {
	TokenID     string `json:"token_id" validate:"required,uuid"`
	Email       string `json:"email" validate:"required,min=5,max=255,email"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=64"`
}

type SigninWithGoogleDto struct {
	Sub     string `json:"sub"` // GoogleID
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}
