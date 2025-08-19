package main

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"reesource-tracker/api"
	"reesource-tracker/lib/database"
	"runtime"
	"strings"

	_ "embed"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	_, devmode := os.LookupEnv("DEV")
	var r *gin.Engine
	if devmode {
		r = gin.Default() // includes Logger and Recovery
	} else {
		gin.SetMode(gin.ReleaseMode)
		r = gin.New() // no Logger
		r.Use(gin.Recovery())
	}
	if devmode {
		println("Running frontend proxy")
		r.Any("/app/*proxypath", proxy)
	} else {
		println("Serving frontend static files")
		r.LoadHTMLGlob("./client/*.html")
		safePath, err := filepath.Abs("./client")
		if err != nil {
			println("Could not resolve client path:", err)
			return
		}
		r.GET("/app/*path", func(c *gin.Context) {
			path := c.Param("path")
			// Check if the first segment is "assets"
			segments := strings.SplitN(path, "/", 3)
			absPath, err := filepath.Abs(filepath.Join(safePath, path))
			if err != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			if absPath != safePath && !strings.HasPrefix(absPath, safePath+string(os.PathSeparator)) {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			if len(segments) > 1 && segments[1] == "assets" {
				c.File("./client" + path)
				return
			}
			c.HTML(http.StatusOK, "index.html", gin.H{})
		})

	}
	database.Connect(context.Background())
	api.Routes(r)
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusPermanentRedirect, "/app")
	})

	if runtime.GOOS == "windows" && devmode {
		r.Run("localhost:80")
	} else {
		r.Run("0.0.0.0:80")
		println("Server started on port 80")
	}
}

func proxy(c *gin.Context) {
	remote, err := url.Parse("http://" + c.Request.Host + ":5173/")
	if err != nil {
		println("Could not resolve Proxy URL")
	}
	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.ServeHTTP(c.Writer, c.Request)
}
