package request

type SignUpDto struct {
	Email       string `json:"email" validate:"required,min=5,max=255,email"`
	Password    string `json:"password" validate:"required,min=8,max=64"`
	DisplayName string `json:"display_name" validate:"required,min=2,max=50"`
}
type SignInDto struct {
	Email    string `json:"email" validate:"required,min=5,max=255,email"`
	Password string `json:"password" validate:"required,min=8,max=64"`
}
