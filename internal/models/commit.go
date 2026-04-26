package models

import (
	"encoding/json"
	"history-api/internal/dtos/response"
	"time"
)

type CommitEntity struct {
	ID           string          `json:"id"`
	ProjectID    string          `json:"project_id"`
	SnapshotJson json.RawMessage `json:"snapshot_json"`
	SnapshotHash string          `json:"snapshot_hash"`
	UserID       string          `json:"user_id"`
	EditSummary  string          `json:"edit_summary"`
	IsDeleted    bool            `json:"is_deleted"`
	CreatedAt    *time.Time       `json:"created_at"`
	User         *UserSimpleEntity `json:"user"`
}

func (c *CommitEntity) ToResponse() *response.CommitResponse {
	if c == nil {
		return nil
	}
	var userRes *response.UserSimpleResponse
	if c.User != nil {
		userRes = c.User.ToResponse()
	}

	return &response.CommitResponse{
		ID:           c.ID,
		ProjectID:    c.ProjectID,
		SnapshotJson: c.SnapshotJson,
		SnapshotHash: c.SnapshotHash,
		UserID:       c.UserID,
		EditSummary:  c.EditSummary,
		CreatedAt:    c.CreatedAt,
		User:         userRes,
	}
}

func CommitsEntityToResponse(entities []*CommitEntity) []*response.CommitResponse {
	res := make([]*response.CommitResponse, 0, len(entities))
	for _, e := range entities {
		res = append(res, e.ToResponse())
	}
	return res
}
