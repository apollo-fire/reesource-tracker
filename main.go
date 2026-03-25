package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"reesource-tracker/api"
	"reesource-tracker/lib/database"
	"runtime"
	"strings"
	"time"

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
			if len(segments) > 1 && segments[1] == "assets" {
				absPath, err := filepath.Abs(filepath.Join(safePath, path))
				if err != nil {
					c.AbortWithStatus(http.StatusInternalServerError)
					return
				}
				if absPath != safePath && !strings.HasPrefix(absPath, safePath+string(os.PathSeparator)) {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
				c.File(absPath)
				return
			}
			c.HTML(http.StatusOK, "index.html", gin.H{})
		})

	}
	// Create context with cancel for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	err := database.Connect(ctx)
	if err != nil {
		log.Fatal("Failed to connect to database:", err.Error())
	}
	api.Routes(r)
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusPermanentRedirect, "/app")
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":80",
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		if runtime.GOOS == "windows" && devmode {
			srv.Addr = "localhost:80"
		}
		println("Server started on", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	<-ctx.Done()

	// Give outstanding requests 5 seconds to complete
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	database.Instance.Close()
	database.Cleanup()

}

func proxy(c *gin.Context) {
	remote, err := url.Parse("http://" + c.Request.Host + ":5173/")
	if err != nil {
		println("Could not resolve Proxy URL")
	}
	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.ServeHTTP(c.Writer, c.Request)
}
