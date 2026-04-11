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

func (t VerifyType) Int16() int16 {
    return int16(t)
}

func ParseVerifyType(v int16) VerifyType {
	switch v {
	case 1:
		return VerifyIdCard
	case 2:
		return VerifyEducation
	case 3:
		return VerifyExpert
	case 4:
		return VerifyOther
	default:
		return VerifyUnknown
	}
}

func ParseVerifyTypeText(v string) VerifyType {
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
