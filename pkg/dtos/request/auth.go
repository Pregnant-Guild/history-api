package request

type SignUpDto struct {
	Password string `json:"password" validate:"required"`
	DiscordUserId string `json:"discord_user_id" validate:"required"`
	Username string `json:"username" validate:"required"`
}

type LoginDto struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

