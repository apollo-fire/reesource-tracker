package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/gin-gonic/gin"
)

func RandomHex(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// HashToken returns the SHA-256 hex digest of the given token. Only the hash
// is stored in the database; the raw token is returned to the caller once at
// creation time.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func DecodeBase64(s string) ([]byte, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err == nil {
		return b, nil
	}
	return base64.StdEncoding.DecodeString(s)
}

func BytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func IsSecureRequest(c *gin.Context) bool {
	// Only allow insecure reqeusts in development mode
	if gin.Mode() == gin.DebugMode {
		if c.Request != nil && c.Request.TLS != nil {
			return true
		}

		xForwardedProto := c.GetHeader("X-Forwarded-Proto")
		for _, proto := range strings.Split(xForwardedProto, ",") {
			if strings.EqualFold(strings.TrimSpace(proto), "https") {
				return true
			}
		}

		return false
	}
	return true
}
