package api

import (
	"reesource-tracker/api/locations"
	"reesource-tracker/api/products"
	"reesource-tracker/api/samples"
	"reesource-tracker/api/sync"

	"github.com/gin-gonic/gin"
)

func Routes(route *gin.Engine) {
	api_routes := route.Group("/api")
	samples.Routes(api_routes)
	products.Routes(api_routes)
	locations.Routes(api_routes)
	sync.Routes(api_routes)
}
