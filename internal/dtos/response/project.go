package response

import "time"

type ProjectResponse struct {
	ID               string              `json:"id"`
	Title            string              `json:"title"`
	Description      string              `json:"description"`
	LatestRevisionID *string             `json:"latest_revision_id,omitempty"`
	VersionCount     int32               `json:"version_count"`
	ProjectStatus    string              `json:"project_status"`
	LockedBy         *string             `json:"locked_by,omitempty"`
	IsDeleted        bool                `json:"is_deleted"`
	UserID           string              `json:"user_id"`
	CreatedAt        *time.Time          `json:"created_at"`
	UpdatedAt        *time.Time          `json:"updated_at"`
	User             *UserSimpleResponse `json:"user,omitempty"`
	CommitIds        []string            `json:"commit_ids"`
	SubmissionIds    []string            `json:"submission_ids"`
}
