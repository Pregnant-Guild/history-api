package models

import (
	"encoding/json"
	"history-api/internal/dtos/response"
	"history-api/pkg/constants"
	"time"
)

type CommitSimple struct {
	ID          string `json:"id"`
	EditSummary string `json:"edit_summary"`
}

type MemberSimple struct {
	UserID      string                      `json:"user_id"`
	Role        constants.ProjectMemberRole `json:"role"`
	DisplayName string                      `json:"display_name"`
	AvatarUrl   string                      `json:"avatar_url"`
}

type ProjectEntity struct {
	ID             string                      `json:"id"`
	Title          string                      `json:"title"`
	Description    string                      `json:"description"`
	LatestCommitID *string                     `json:"latest_commit_id"`
	ProjectStatus  constants.ProjectStatusType `json:"project_status"`
	LockedBy       *string                     `json:"locked_by"`
	IsDeleted      bool                        `json:"is_deleted"`
	UserID         string                      `json:"user_id"`
	CreatedAt      *time.Time                  `json:"created_at"`
	UpdatedAt      *time.Time                  `json:"updated_at"`
	User           *UserSimpleEntity           `json:"user"`
	Commits        []CommitSimple              `json:"commits"`
	SubmissionIds  []string                    `json:"submission_ids"`
	Members        []MemberSimple              `json:"members"`
}

func (p *ProjectEntity) ParseUser(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		p.User = nil
		return nil
	}
	return json.Unmarshal(data, &p.User)
}

func (p *ProjectEntity) ParseCommits(data []byte) error {
	if len(data) == 0 || string(data) == "null" || string(data) == "[]" {
		p.Commits = []CommitSimple{}
		return nil
	}
	return json.Unmarshal(data, &p.Commits)
}

func (p *ProjectEntity) ParseMembers(data []byte) error {
	if len(data) == 0 || string(data) == "null" || string(data) == "[]" {
		p.Members = []MemberSimple{}
		return nil
	}
	return json.Unmarshal(data, &p.Members)
}

func (p *ProjectEntity) ToResponse() *response.ProjectResponse {
	if p == nil {
		return nil
	}
	var userResponse *response.UserSimpleResponse
	if p.User != nil {
		userResponse = p.User.ToResponse()
	}

	commits := make([]response.CommitSimpleResponse, 0, len(p.Commits))
	for _, c := range p.Commits {
		commits = append(commits, response.CommitSimpleResponse{
			ID:          c.ID,
			EditSummary: c.EditSummary,
		})
	}

	members := make([]response.MemberSimpleResponse, 0, len(p.Members))
	for _, m := range p.Members {
		members = append(members, response.MemberSimpleResponse{
			UserID:      m.UserID,
			Role:        m.Role.String(),
			DisplayName: m.DisplayName,
			AvatarUrl:   m.AvatarUrl,
		})
	}

	return &response.ProjectResponse{
		ID:             p.ID,
		Title:          p.Title,
		Description:    p.Description,
		LatestCommitID: p.LatestCommitID,
		ProjectStatus:  p.ProjectStatus.String(),
		LockedBy:       p.LockedBy,
		IsDeleted:      p.IsDeleted,
		UserID:         p.UserID,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
		User:           userResponse,
		Commits:        commits,
		SubmissionIds:  p.SubmissionIds,
		Members:        members,
	}
}

func ProjectsEntityToResponse(projects []*ProjectEntity) []*response.ProjectResponse {
	out := make([]*response.ProjectResponse, 0)
	if projects == nil {
		return out
	}
	for _, project := range projects {
		if project == nil {
			continue
		}
		out = append(out, project.ToResponse())
	}
	return out
}
