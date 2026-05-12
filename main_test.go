package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestFrontendHandler_ServesAssetFile(t *testing.T) {
	router, _ := setupFrontendTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/app/assets/app.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "console.log('ok');", strings.TrimSpace(rec.Body.String()))
}

func TestFrontendHandler_NonAssetPathReturnsIndex(t *testing.T) {
	router, _ := setupFrontendTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/app/secret.txt", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "test index")
}

func TestFrontendHandler_BackslashPathIsForbidden(t *testing.T) {
	router, _ := setupFrontendTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/app/assets/foo%5Cbar.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestFrontendHandler_TraversalPathIsForbidden(t *testing.T) {
	router, _ := setupFrontendTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/app/assets/../../secret.txt", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestLegacyHandler_ProofOfConcept_DirectNonAssetFileAccess(t *testing.T) {
	router, _ := setupFrontendTestRouterWithHandler(t, legacyNoAssetScopeHandler)

	req := httptest.NewRequest(http.MethodGet, "/app/secret.txt", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "should not be served directly", strings.TrimSpace(rec.Body.String()))
}

func TestLegacyHandler_ProofOfConcept_PathTraversalAttack(t *testing.T) {
	clientDir := setupFrontendFixture(t)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/app/assets/app.js", nil)
	c.Params = gin.Params{{Key: "path", Value: "/assets/../secret.txt"}}

	legacyNoCleanScopeCheckHandler(clientDir)(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "should not be served directly", strings.TrimSpace(rec.Body.String()))
}

func setupFrontendTestRouter(t *testing.T) (*gin.Engine, string) {
	return setupFrontendTestRouterWithHandler(t, frontendHandler)
}

func setupFrontendTestRouterWithHandler(t *testing.T, handlerFactory func(string) gin.HandlerFunc) (*gin.Engine, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	clientDir := setupFrontendFixture(t)

	router := gin.New()
	router.LoadHTMLGlob(filepath.Join(clientDir, "*.html"))
	router.GET("/app/*path", handlerFactory(clientDir))

	return router, clientDir
}

func setupFrontendFixture(t *testing.T) string {
	t.Helper()

	clientDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(clientDir, "index.html"), []byte("<html><body>test index</body></html>"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(clientDir, "assets"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(clientDir, "assets", "app.js"), []byte("console.log('ok');"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(clientDir, "secret.txt"), []byte("should not be served directly"), 0o644))
	return clientDir
}

func legacyNoAssetScopeHandler(baseDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Param("path")
		relPath := strings.TrimPrefix(path, "/")
		absPath, err := filepath.Abs(filepath.Join(baseDir, relPath))
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if absPath != baseDir && !strings.HasPrefix(absPath, baseDir+string(os.PathSeparator)) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.File(absPath)
	}
}

func legacyNoCleanScopeCheckHandler(baseDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Param("path")
		if strings.HasPrefix(path, "/assets/") {
			relPath := strings.TrimPrefix(path, "/")
			absPath, err := filepath.Abs(filepath.Join(baseDir, relPath))
			if err != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			if absPath != baseDir && !strings.HasPrefix(absPath, baseDir+string(os.PathSeparator)) {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.File(absPath)
			return
		}
		c.HTML(http.StatusOK, "index.html", gin.H{})
	}
}
