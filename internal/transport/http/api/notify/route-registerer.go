package notify

import (
	"github.com/D1sordxr/delayed-notifier/internal/transport/http/api/notify/handler"

	"github.com/gin-gonic/gin"
	"github.com/wb-go/wbf/ginext"
)

type RouteRegisterer struct {
	handlers    *handler.Handlers
	middlewares []gin.HandlerFunc
}

func NewRouteRegisterer(
	handlers *handler.Handlers,
	middlewares ...gin.HandlerFunc,
) *RouteRegisterer {
	return &RouteRegisterer{
		handlers:    handlers,
		middlewares: middlewares,
	}
}

func (r *RouteRegisterer) RegisterRoutes(router *ginext.RouterGroup) {
	notifyGroup := router.RouterGroup
	for _, mw := range r.middlewares {
		notifyGroup.Use(mw)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "OK"})
	})

	handler.RegisterHandlers(
		notifyGroup,
		handler.NewStrictHandler(
			r.handlers,
			[]handler.StrictMiddlewareFunc{},
		),
	)
}
