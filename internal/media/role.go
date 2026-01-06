package media

import (
	"fmt"
)

type Role int

const (
	RoleUnknown Role = iota
	RoleCaller
	RoleCallee
)

func (role Role) String() string {
	switch role {
	case RoleUnknown:
		return "unknown"
	case RoleCaller:
		return "caller"
	case RoleCallee:
		return "callee"
	default:
		return "unknown"

	}
}

func GetRole(role string) (Role, error) {
	switch role {
	case "caller":
		return RoleCaller, nil
	case "callee":
		return RoleCallee, nil
	default:
		return 0, fmt.Errorf("Invalid Role: Role must be caller or callee", role)
	}
}
