package response

type RoleSimpleResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RoleResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	IsDeleted bool    `json:"is_deleted"`
	CreatedAt *string `json:"created_at"`
	UpdatedAt *string `json:"updated_at"`
}