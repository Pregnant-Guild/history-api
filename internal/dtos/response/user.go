package response

import "time"

type UserResponse struct {
	ID           string                     `json:"id"`
	Email        string                     `json:"email"`
	Profile      *UserProfileSimpleResponse `json:"profile"`
	TokenVersion int32                      `json:"token_version"`
	IsDeleted    bool                       `json:"is_deleted"`
	CreatedAt    *time.Time                 `json:"created_at"`
	UpdatedAt    *time.Time                 `json:"updated_at"`
	Roles        []*RoleSimpleResponse      `json:"roles"`
}

type UserSimpleResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	FullName    string `json:"full_name"`
	AvatarUrl   string `json:"avatar_url"`
}

type UserProfileSimpleResponse struct {
	DisplayName string `json:"display_name"`
	FullName    string `json:"full_name"`
	AvatarUrl   string `json:"avatar_url"`
	Bio         string `json:"bio"`
	Location    string `json:"location"`
	Website     string `json:"website"`
	CountryCode string `json:"country_code"`
	Phone       string `json:"phone"`
}
