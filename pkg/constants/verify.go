package constants

type VerifyType int16

const (
	VerifyTypeUnknown   VerifyType = 0
	VerifyTypeIdCard    VerifyType = 1
	VerifyTypeEducation VerifyType = 2
	VerifyTypeExpert    VerifyType = 3
	VerifyTypeOther     VerifyType = 4
)

func (t VerifyType) String() string {
	switch t {
	case VerifyTypeIdCard:
		return "ID_CARD"
	case VerifyTypeEducation:
		return "EDUCATION"
	case VerifyTypeExpert:
		return "EXPERT"
	case VerifyTypeOther:
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
		return VerifyTypeIdCard
	case 2:
		return VerifyTypeEducation
	case 3:
		return VerifyTypeExpert
	case 4:
		return VerifyTypeOther
	default:
		return VerifyTypeUnknown
	}
}

func ParseVerifyTypeText(v string) VerifyType {
	switch v {
	case "ID_CARD":
		return VerifyTypeIdCard
	case "EDUCATION":
		return VerifyTypeEducation
	case "EXPERT":
		return VerifyTypeExpert
	case "OTHER":
		return VerifyTypeOther
	default:
		return VerifyTypeUnknown
	}
}
