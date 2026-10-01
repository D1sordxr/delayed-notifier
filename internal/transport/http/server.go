package http

import (
	"net/http"
	"time"

	"github.com/D1sordxr/delayed-notifier/internal/infra/config"
	"github.com/D1sordxr/delayed-notifier/internal/transport/http/middleware"

	"github.com/gin-gonic/gin"
)

type routeRegisterer interface {
	RegisterRoutes(router *gin.RouterGroup)
}

// NewHandler builds the API router: common middleware, then each
// registerer's routes under /api.
func NewHandler(cfg *config.HTTPServer, registerers ...routeRegisterer) http.Handler {
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()
	engine.Use(middleware.Logger())
	engine.Use(middleware.Recovery())

	if cfg.CORS {
		allowedOrigins := cfg.AllowOrigins
		if len(allowedOrigins) == 0 {
			allowedOrigins = []string{"*"}
		}

		engine.Use(middleware.CORS(middleware.CORSConfig{
			AllowOrigins:     allowedOrigins,
			AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))
	}

	api := engine.Group("/api")
	for _, r := range registerers {
		r.RegisterRoutes(api)
	}

	return engine.Handler()
}
