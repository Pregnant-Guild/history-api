package constants

type ProjectMemberRole int16

const (
	ProjectMemberRoleOwner  ProjectMemberRole = 1
	ProjectMemberRoleEditor ProjectMemberRole = 2
	ProjectMemberRoleViewer ProjectMemberRole = 3
)

func (r ProjectMemberRole) String() string {
	switch r {
	case ProjectMemberRoleOwner:
		return "OWNER"
	case ProjectMemberRoleEditor:
		return "EDITOR"
	case ProjectMemberRoleViewer:
		return "VIEWER"
	default:
		return "UNKNOWN"
	}
}

func (r ProjectMemberRole) Int16() int16 {
	return int16(r)
}

func ParseProjectMemberRole(v int16) ProjectMemberRole {
	switch v {
	case 1:
		return ProjectMemberRoleOwner
	case 2:
		return ProjectMemberRoleEditor
	case 3:
		return ProjectMemberRoleViewer
	default:
		return ProjectMemberRoleViewer
	}
}

func ParseProjectMemberRoleText(v string) ProjectMemberRole {
	switch v {
	case "OWNER":
		return ProjectMemberRoleOwner
	case "EDITOR":
		return ProjectMemberRoleEditor
	case "VIEWER":
		return ProjectMemberRoleViewer
	default:
		return ProjectMemberRoleViewer
	}
}

func (r ProjectMemberRole) CanWrite() bool {
	return r == ProjectMemberRoleOwner || r == ProjectMemberRoleEditor
}

func (r ProjectMemberRole) CanRead() bool {
	return r == ProjectMemberRoleOwner || r == ProjectMemberRoleEditor || r == ProjectMemberRoleViewer
}

func (r ProjectMemberRole) IsOwner() bool {
	return r == ProjectMemberRoleOwner
}
