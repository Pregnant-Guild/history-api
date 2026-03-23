package request

import "history-api/pkg/constant"

type CreateUserDto struct {
	Username      string          `json:"username" validate:"required"`
	Password      string          `json:"password" validate:"required"`
	DiscordUserId string          `json:"discord_user_id" validate:"required"`
	Role          []constant.Role `json:"role" validate:"required"`
}

type UpdateUserDto struct {
	Password      *string          `json:"password" validate:"omitempty"`
	DiscordUserId *string          `json:"discord_user_id" validate:"omitempty"`
	Role          *[]constant.Role `json:"role" validate:"omitempty"`
}

type SearchUserDto struct {
	Username      *string          `query:"username" validate:"omitempty"`
	DiscordUserId *string          `query:"discord_user_id" validate:"omitempty"`
	Role          *[]constant.Role `query:"role" validate:"omitempty"`
	SortBy        string           `query:"sort_by" default:"created_at" validate:"oneof=created_at updated_at"`
	Order         string           `query:"order" default:"desc" validate:"oneof=asc desc"`
	Page          int              `query:"page" default:"1" validate:"min=1"`
	Limit         int              `query:"limit" default:"10" validate:"min=1,max=100"`
}
