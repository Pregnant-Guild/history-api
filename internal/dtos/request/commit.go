package request

import "encoding/json"

type CreateCommitDto struct {
	SnapshotJson json.RawMessage `json:"snapshot_json" validate:"required"`
	EditSummary  string          `json:"edit_summary" validate:"required,max=500"`
}

type RestoreCommitDto struct {
	CommitID string `json:"commit_id" validate:"required,uuid"`
}
