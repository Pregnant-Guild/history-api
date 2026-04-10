package constants

type VerifyType int16

const (
	VerifyUnknown   VerifyType = 0
	VerifyIdCard    VerifyType = 1
	VerifyEducation VerifyType = 2
	VerifyExpert    VerifyType = 3
	VerifyOther     VerifyType = 4
)

func (t VerifyType) String() string {
	switch t {
	case VerifyIdCard:
		return "ID_CARD"
	case VerifyEducation:
		return "EDUCATION"
	case VerifyExpert:
		return "EXPERT"
	case VerifyOther:
		return "OTHER"
	default:
		return "UNKNOWN"
	}
}

func ParseVerifyType(v string) VerifyType {
	switch v {
	case "ID_CARD":
		return VerifyIdCard
	case "EDUCATION":
		return VerifyEducation
	case "EXPERT":
		return VerifyExpert
	case "OTHER":
		return VerifyOther
	default:
		return VerifyUnknown
	}
}
