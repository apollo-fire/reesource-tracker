package auth

const (
	RoleAdmin      = "admin"
	RoleMaintainer = "maintainer"
	RoleUser       = "user"
)

var AllowedRoles = []string{RoleAdmin, RoleMaintainer, RoleUser}

func IsValidRole(role string) bool {
	for _, r := range AllowedRoles {
		if r == role {
			return true
		}
	}
	return false
}

func HasRole(roles []string, required string) bool {
	for _, r := range roles {
		if r == required {
			return true
		}
	}
	return false
}
