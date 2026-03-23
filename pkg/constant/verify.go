package constant

type VerifyType int16

const (
	VerifyIdCard    VerifyType = 1
	VerifyEducation VerifyType = 2
	VerifyExpert    VerifyType = 3
)

func (t VerifyType) String() string {
	switch t {
	case VerifyIdCard:
		return "ID_CARD"
	case VerifyEducation:
		return "EDUCATION"
	case VerifyExpert:
		return "EXPERT"
	default:
		return "UNKNOWN"
	}
}

func ParseVerifyType(v int16) VerifyType {
	switch v {
	case 1:
		return VerifyIdCard
	case 2:
		return VerifyEducation
	case 3:
		return VerifyExpert
	default:
		return 0
	}
}
