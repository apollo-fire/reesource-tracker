package api

import (
	"reesource-tracker/api/auth"
	"reesource-tracker/api/locations"
	"reesource-tracker/api/middleware"
	"reesource-tracker/api/products"
	"reesource-tracker/api/samples"
	"reesource-tracker/api/sync"
	"reesource-tracker/api/users"
	libauth "reesource-tracker/lib/auth"

	"github.com/gin-gonic/gin"
)

func Routes(route *gin.Engine) {
	api := route.Group("/api")

	// Auth routes are public (login redirect, callback, logout, logout channels).
	auth.Routes(api)

	// All application routes require an authenticated session.
	protected := api.Group("")
	protected.Use(middleware.RequireAuthenticated())
	products.ReadRoutes(protected)
	locations.ReadRoutes(protected)
	samples.Routes(protected)
	sync.Routes(protected)

	// Maintainer+ routes: product/location management and sample code provisioning.
	maintainer := protected.Group("")
	maintainer.Use(middleware.RequireRole(libauth.RoleMaintainer))
	products.MaintainerRoutes(maintainer)
	locations.MaintainerRoutes(maintainer)
	samples.MaintainerRoutes(maintainer)

	// Admin-only routes: user management.
	admin := protected.Group("")
	admin.Use(middleware.RequireRole(libauth.RoleAdmin))
	users.Routes(admin)
}
