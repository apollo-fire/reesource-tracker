package oidc

import (
	"encoding/json"
	"strings"

	libauth "reesource-tracker/lib/auth"
)

// ExtractRoles parses rawClaims using a dot-notation claimPath (e.g.
// "groups" or "realm_access.roles"), maps each value through roleMap, and
// returns the matching app roles. If roleMap is empty, claim values are
// matched directly against known role names.
func ExtractRoles(rawClaims json.RawMessage, claimPath string, roleMap map[string]string) []string {
	var claims map[string]any
	if err := json.Unmarshal(rawClaims, &claims); err != nil {
		return nil
	}

	var current any = claims
	for _, part := range strings.Split(claimPath, ".") {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[part]
	}

	var oidcValues []string
	switch v := current.(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				oidcValues = append(oidcValues, s)
			}
		}
	case []string:
		oidcValues = v
	}

	if len(roleMap) == 0 {
		var roles []string
		for _, val := range oidcValues {
			if libauth.IsValidRole(val) {
				roles = append(roles, val)
			}
		}
		return roles
	}

	seen := map[string]bool{}
	var roles []string
	for _, val := range oidcValues {
		if appRole, ok := roleMap[val]; ok && libauth.IsValidRole(appRole) && !seen[appRole] {
			roles = append(roles, appRole)
			seen[appRole] = true
		}
	}
	return roles
}
