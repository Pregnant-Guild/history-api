package constants

type ProjectStatusType int16


const (
	ProjectStatusTypePrivate ProjectStatusType = 1
	ProjectStatusTypePublic ProjectStatusType = 2
	ProjectStatusTypeArchive ProjectStatusType = 3
	ProjectStatusTypeUnknow ProjectStatusType = 0
)

func (t ProjectStatusType) String() string {
	switch t {
	case ProjectStatusTypePrivate:
		return "PRIVATE"
	case ProjectStatusTypePublic:
		return "PUBLIC"
	case ProjectStatusTypeArchive:
		return "ARCHIVE"
	default:
		return "UNKNOWN"
	}
}

func ParseProjectStatusTypeText(v string) ProjectStatusType {
	switch v {
	case "PRIVATE":
		return ProjectStatusTypePrivate
	case "PUBLIC":
		return ProjectStatusTypePublic
	case "ARCHIVE":
		return ProjectStatusTypeArchive
	default:
		return ProjectStatusTypeUnknow
	}
}

func (t ProjectStatusType) Int16() int16 {
    return int16(t)
}

func ParseProjectStatusType(v int16) ProjectStatusType {
	switch v {
	case 1:
		return ProjectStatusTypePrivate
	case 2:
		return ProjectStatusTypePublic
	case 3:
		return ProjectStatusTypeArchive
	default:
		return ProjectStatusTypeUnknow
	}
}