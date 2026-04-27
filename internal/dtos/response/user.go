package response

import "time"

type UserResponse struct {
	ID           string                     `json:"id"`
	Email        string                     `json:"email"`
	Profile      *UserProfileSimpleResponse `json:"profile,omitempty"`
	TokenVersion int32                      `json:"token_version,omitempty"`
	IsDeleted    bool                       `json:"is_deleted,omitempty"`
	AuthProvider string                     `json:"auth_provider,omitempty"`
	GoogleID     string                     `json:"google_id,omitempty"`
	CreatedAt    *time.Time                 `json:"created_at,omitempty"`
	UpdatedAt    *time.Time                 `json:"updated_at,omitempty"`
	Roles        []*RoleSimpleResponse      `json:"roles,omitempty"`
}

type UserSimpleResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name,omitempty"`
	FullName    string `json:"full_name,omitempty"`
	AvatarUrl   string `json:"avatar_url,omitempty"`
}

type UserProfileSimpleResponse struct {
	DisplayName string `json:"display_name"`
	FullName    string `json:"full_name,omitempty"`
	AvatarUrl   string `json:"avatar_url,omitempty"`
	Bio         string `json:"bio,omitempty"`
	Location    string `json:"location,omitempty"`
	Website     string `json:"website,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	Phone       string `json:"phone,omitempty"`
}
