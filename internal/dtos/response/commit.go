package response

import (
	"encoding/json"
	"time"
)

type CommitResponse struct {
	ID           string              `json:"id"`
	ProjectID    string              `json:"project_id"`
	SnapshotJson json.RawMessage     `json:"snapshot_json"`
	SnapshotHash string              `json:"snapshot_hash"`
	UserID       string              `json:"user_id"`
	EditSummary  string              `json:"edit_summary"`
	CreatedAt    *time.Time          `json:"created_at"`
	User         *UserSimpleResponse `json:"user,omitempty"`
}
