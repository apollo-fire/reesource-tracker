package products_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"reesource-tracker/api/products"
	"reesource-tracker/lib/database"
	"reesource-tracker/lib/test_helpers/mock_db"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func routeMethodsForPath(routes gin.RoutesInfo, path string) []string {
	var methods []string
	for _, route := range routes {
		if route.Path == path {
			methods = append(methods, route.Method)
		}
	}
	slices.Sort(methods)
	return methods
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	group := r.Group("/api")
	products.Routes(group)
	return r
}

func TestReadRoutes_RegisterOnlyReadEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	products.ReadRoutes(r.Group("/api"))

	routes := r.Routes()
	assert.Equal(t, []string{"GET"}, routeMethodsForPath(routes, "/api/products"))
	assert.Equal(t, []string{"GET"}, routeMethodsForPath(routes, "/api/product/:product_id"))
	assert.Empty(t, routeMethodsForPath(routes, "/api/product"))
}

func TestMaintainerRoutes_RegisterOnlyMutationEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	products.MaintainerRoutes(r.Group("/api"))

	routes := r.Routes()
	assert.Equal(t, []string{"POST"}, routeMethodsForPath(routes, "/api/product"))
	assert.Equal(t, []string{"DELETE", "POST"}, routeMethodsForPath(routes, "/api/product/:product_id"))
	assert.Empty(t, routeMethodsForPath(routes, "/api/products"))
}

func TestCreateProduct_Success(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	body := map[string]interface{}{"name": "Test Product"}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/product", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

func TestGetProduct_NotFound(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/product/00000000-0000-0000-0000-000000000000", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestUpdateProduct_InvalidID(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	body := map[string]string{"name": "Updated Product"}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/product/not-a-uuid", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestDeleteProduct_MissingID(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/product/", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func TestDeleteProduct_InvalidID(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/product/not-a-uuid", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestDeleteProduct_Success(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/product/123e4567-e89b-12d3-a456-426614174000", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "deleted")
}
