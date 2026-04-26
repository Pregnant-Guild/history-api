package request

type AddProjectMemberDto struct {
	UserID string `json:"user_id" validate:"required,uuid"`
	Role   string `json:"role" validate:"required,oneof=EDITOR VIEWER"`
}

type UpdateProjectMemberDto struct {
	Role string `json:"role" validate:"required,oneof=EDITOR VIEWER"`
}

type ChangeOwnerDto struct {
	NewOwnerID string `json:"new_owner_id" validate:"required,uuid"`
}
