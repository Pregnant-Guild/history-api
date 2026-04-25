package constants

type RoleType string

const (
	RoleTypeAdmin     RoleType = "ADMIN"
	RoleTypeMod       RoleType = "MOD"
	RoleTypeUser      RoleType = "USER"
	RoleTypeHistorian RoleType = "HISTORIAN"
	RoleTypeBanned    RoleType = "BANNED"
)

func (r RoleType) String() string {
	return string(r)
}

func (r RoleType) Compare(other RoleType) bool {
	return r == other
}

func (r RoleType) IsValid() bool {
	return CheckValidRole(r)
}

func CheckValidRole(r RoleType) bool {
	return r == RoleTypeAdmin || r == RoleTypeMod || r == RoleTypeHistorian || r == RoleTypeUser || r == RoleTypeBanned
}


func ParseRole(s string) (RoleType, bool) {
	r := RoleType(s)
	if CheckValidRole(r) {
		return r, true
	}
	return "", false
}
func (r RoleType) ToSlice() []RoleType {
	return []RoleType{r}
}
