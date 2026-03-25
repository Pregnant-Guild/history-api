package constant

type Role string

const (
	ADMIN     Role = "ADMIN"
	MOD       Role = "MOD"
	USER      Role = "USER"
	HISTORIAN Role = "HISTORIAN"
	BANNED    Role = "BANNED"
)

func (r Role) String() string {
	return string(r)
}

func (r Role) Compare(other Role) bool {
	return r == other
}

func CheckValidRole(r Role) bool {
	return r == ADMIN || r == MOD || r == HISTORIAN || r == USER || r == BANNED
}

func ParseRole(s string) (Role, bool) {
	r := Role(s)
	if CheckValidRole(r) {
		return r, true
	}
	return "", false
}
func (r Role) ToSlice() []Role {
	return []Role{r}
}
