package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
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
		r = gin.New() // no Logger
		r.Use(gin.Recovery())
	}
	if devmode {
		println("Running frontend proxy")
		r.Any("/app/*proxypath", proxy)
	} else {
		println("Serving frontend static files")
		r.LoadHTMLGlob("./client/*.html")
		r.GET("/app/*path", func(c *gin.Context) {
			path := c.Param("path")
			// Check if the first segment is "assets"
			segments := strings.SplitN(path, "/", 3)
			if len(segments) > 1 && segments[1] == "assets" {
				c.File("./client" + path)
				return
			}
			c.HTML(http.StatusOK, "index.html", gin.H{})
		})

	}
	err := database.Connect(context.Background())
	if err != nil {
		log.Fatal("Failed to connect to database:", err.Error())
	}
	api.Routes(r)
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusPermanentRedirect, "/app")
	})

	if runtime.GOOS == "windows" && devmode {
		r.Run("localhost:80")
	} else {
		r.Run("0.0.0.0:80")
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
