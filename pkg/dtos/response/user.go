package response

type UserResponse struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Email        string        `json:"email"`
	AvatarUrl    string        `json:"avatar_url"`
	IsActive     bool          `json:"is_active"`
	IsVerified   bool          `json:"is_verified"`
	TokenVersion int32         `json:"token_version"`
	IsDeleted    bool          `json:"is_deleted"`
	CreatedAt    *string       `json:"created_at"`
	UpdatedAt    *string       `json:"updated_at"`
	Roles        []*RoleSimpleResponse  `json:"roles"`
}