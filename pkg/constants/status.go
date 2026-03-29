package constants

type StatusType int16

const (
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
		return "REJECT"
	default:
		return "UNKNOWN"
	}
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
		return 0
	}
}
