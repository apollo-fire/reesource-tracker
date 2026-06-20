package auth

const (
	RoleAdmin      = "admin"
	RoleMaintainer = "maintainer"
	RoleUser       = "user"
)

var AllowedRoles = []string{RoleAdmin, RoleMaintainer, RoleUser}

func IsValidRole(role string) bool {
	for _, candidate := range AllowedRoles {
		if candidate == role {
			return true
		}
	}
	return false
}

func HasRole(roles []string, role string) bool {
	for _, candidate := range roles {
		if candidate == role {
			return true
		}
	}
	return false
}
