package response

import "time"

type CommitSimpleResponse struct {
	ID          string `json:"id"`
	EditSummary string `json:"edit_summary"`
}

type MemberSimpleResponse struct {
	UserID      string `json:"user_id"`
	Role        string `json:"role"`
	DisplayName string `json:"display_name"`
	AvatarUrl   string `json:"avatar_url"`
}

type SubmissionSimpleResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type ProjectResponse struct {
	ID             string                     `json:"id"`
	Title          string                     `json:"title"`
	Description    string                     `json:"description"`
	LatestCommitID *string                    `json:"latest_commit_id,omitempty"`
	ProjectStatus  string                     `json:"project_status"`
	LockedBy       *string                    `json:"locked_by,omitempty"`
	IsDeleted      bool                       `json:"is_deleted"`
	UserID         string                     `json:"user_id"`
	CreatedAt      *time.Time                 `json:"created_at"`
	UpdatedAt      *time.Time                 `json:"updated_at"`
	User           *UserSimpleResponse        `json:"user,omitempty"`
	Commits        []CommitSimpleResponse     `json:"commits"`
	Submissions    []SubmissionSimpleResponse `json:"submissions"`
	Members        []MemberSimpleResponse     `json:"members"`
}
