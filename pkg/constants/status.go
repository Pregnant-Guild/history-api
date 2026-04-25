package constants

type StatusType int16

const (
	StatusTypeUnknown  StatusType = 0
	StatusTypePending  StatusType = 1
	StatusTypeApproved StatusType = 2
	StatusTypeRejected StatusType = 3
)

func (t StatusType) String() string {
	switch t {
	case StatusTypePending:
		return "PENDING"
	case StatusTypeApproved:
		return "APPROVED"
	case StatusTypeRejected:
		return "REJECTED"
	default:
		return "UNKNOWN"
	}
}

func (t StatusType) Int16() int16 {
    return int16(t)
}

func ParseStatusType(v int16) StatusType {
	switch v {
	case 1:
		return StatusTypePending
	case 2:
		return StatusTypeApproved
	case 3:
		return StatusTypeRejected
	default:
		return StatusTypeUnknown
	}
}

func ParseStatusTypeText(v string) StatusType {
	switch v {
	case "PENDING":
		return StatusTypePending
	case "APPROVED":
		return StatusTypeApproved
	case "REJECTED":
		return StatusTypeRejected
	default:
		return StatusTypeUnknown
	}
}
