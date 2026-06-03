package auth

import (
	"bytes"
	"encoding/hex"
	"net/http"
	"net/url"

	"reesource-tracker/lib/database"
)

// DecodeCredentialID parses a credential ID from a raw string that may be
// hex-encoded or base64url-encoded. Returns the raw bytes and the canonical
// hex string.
func DecodeCredentialID(raw string) ([]byte, string, error) {
	id, err := hex.DecodeString(raw)
	if err == nil {
		return id, raw, nil
	}
	id, err = DecodeBase64(raw)
	if err != nil {
		return nil, "", err
	}
	return id, hex.EncodeToString(id), nil
}

// BuildAssignmentURL constructs the full assignment link URL from an incoming
// HTTP request and the raw token. It respects X-Forwarded-Proto and
// X-Forwarded-Host proxy headers.
func BuildAssignmentURL(r *http.Request, token string) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return scheme + "://" + host + "/app?assignment_token=" + url.QueryEscape(token)
}

// AssignmentLinkMap converts a PasskeyAssignmentLink row to a response map.
// If rawToken is non-empty, the token and a full assignment URL are included.
func AssignmentLinkMap(r *http.Request, row database.PasskeyAssignmentLink, rawToken string) map[string]any {
	res := map[string]any{
		"link_id": row.ID,
	}
	if rawToken != "" {
		res["assignment_token"] = rawToken
		res["assignment_url"] = BuildAssignmentURL(r, rawToken)
	}
	if row.ExpiresAt.Valid {
		res["expires_at"] = row.ExpiresAt.Time
	}
	return res
}

// PasskeyListMap converts passkey rows to response maps, filtering out revoked
// entries and marking the active session credential.
func PasskeyListMap(rows []database.Passkey, activeCredentialID []byte) []map[string]any {
	res := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if row.RevokedAt.Valid {
			continue
		}
		label := ""
		if row.Label.Valid {
			label = row.Label.String
		}
		res = append(res, map[string]any{
			"credential_id":      hex.EncodeToString(row.CredentialID),
			"label":              label,
			"created_at":         row.CreatedAt,
			"is_current_session": len(activeCredentialID) > 0 && bytes.Equal(row.CredentialID, activeCredentialID),
		})
	}
	return res
}
