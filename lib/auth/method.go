package auth

import "github.com/gin-gonic/gin"

// AuthMethod represents a pluggable sign-in credential type (e.g. passkeys,
// email magic links). Each method registers its own unauthenticated sign-in
// routes, authenticated self-management routes, and admin management routes.
//
// Implementations live under api/auth/<method>/ so that each auth method is
// fully self-contained. The central auth.Routes() wiring file calls these
// methods in sequence; adding a new auth method only requires implementing
// this interface and registering it there.
type AuthMethod interface {
	// Routes registers publicly accessible endpoints – sign-in flows that
	// do not require an existing session (e.g. login/begin, login/consume).
	Routes(authRoutes *gin.RouterGroup)

	// SelfRoutes registers authenticated endpoints that allow the currently
	// signed-in user to manage their own credentials (list, add, remove).
	SelfRoutes(self *gin.RouterGroup)

	// AdminRoutes registers admin-only endpoints for managing any user's
	// credentials.
	AdminRoutes(admin *gin.RouterGroup)
}
