package consts

type UserRole uint

const (
	Guest UserRole = iota + 1
	User
	Admin
	Operator
)

func (r UserRole) String() string {
	switch r {
	case Guest:
		return "guest"
	case User:
		return "user"
	case Admin:
		return "admin"
	case Operator:
		return "operator"
	default:
		return "unknown"
	}
}
