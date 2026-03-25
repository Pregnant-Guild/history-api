package response

import "time"

type RoleSimpleResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RoleResponse struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	IsDeleted bool       `json:"is_deleted"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}
