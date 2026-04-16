package constants

type StatusType int16

const (
	StatusUnknown  StatusType = 0
	StatusPending  StatusType = 1
	StatusApproved StatusType = 2
	StatusRejected StatusType = 3
)

func (t StatusType) String() string {
	switch t {
	case StatusPending:
		return "PENDING"
	case StatusApproved:
		return "APPROVED"
	case StatusRejected:
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
		return StatusPending
	case 2:
		return StatusApproved
	case 3:
		return StatusRejected
	default:
		return StatusUnknown
	}
}

func ParseStatusTypeText(v string) StatusType {
	switch v {
	case "PENDING":
		return StatusPending
	case "APPROVED":
		return StatusApproved
	case "REJECTED":
		return StatusRejected
	default:
		return StatusUnknown
	}
}
