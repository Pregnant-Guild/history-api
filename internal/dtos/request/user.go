package request

type UpdateProfileDto struct {
	DisplayName string `json:"display_name" validate:"omitempty,min=2,max=50"`
	FullName    string `json:"full_name" validate:"omitempty,min=2,max=100"`
	AvatarUrl   string `json:"avatar_url" validate:"omitempty,url"`
	Bio         string `json:"bio" validate:"omitempty,max=255"`
	Location    string `json:"location" validate:"omitempty,max=100"`
	Website     string `json:"website" validate:"omitempty,url"`
	CountryCode string `json:"country_code" validate:"omitempty,len=2"`
	Phone       string `json:"phone" validate:"omitempty,min=8,max=20"`
}

type ChangePasswordDto struct {
	OldPassword string `json:"old_password" validate:"required,min=8,max=64"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=64,nefield=OldPassword"`
}

type ChangeRoleDto struct {
	UserID string   `json:"user_id" validate:"required,uuid"`
	Roles  []string `json:"role_ids" validate:"required,min=1,dive,required,uuid"`
}

type GetAllUserDto struct {
	CursorPaginationDto
	IsDeleted *bool    `json:"is_deleted" query:"is_deleted" validate:"omitempty"`
	RoleIDs   []string `json:"role_ids" query:"role_ids" validate:"omitempty,dive,uuid"`
}

type CursorPaginationDto struct {
	Cursor string `json:"cursor" query:"cursor" validate:"omitempty,uuid"`
	Limit  int    `json:"limit" query:"limit" validate:"required,min=1,max=100"`
	Sort   string `json:"sort" query:"sort" validate:"omitempty,oneof=id created_at updated_at"`
	Order  string `json:"order" query:"order" validate:"omitempty,oneof=asc desc"`
}
type SearchUserDto struct {
	CursorPaginationDto
	Search    string   `json:"search" query:"search" validate:"omitempty,min=2,max=200"`
	IsDeleted *bool    `json:"is_deleted" query:"is_deleted" validate:"omitempty"`
	RoleIDs   []string `json:"role_ids" query:"role_ids" validate:"omitempty,dive,uuid"`
}
