package notify

import (
	"github.com/D1sordxr/delayed-notifier/internal/transport/http/api/notify/handler"

	"github.com/gin-gonic/gin"
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

func (r *RouteRegisterer) RegisterRoutes(router *gin.RouterGroup) {
	router.Use(r.middlewares...)

	handler.RegisterHandlers(
		router,
		handler.NewStrictHandler(r.handlers, nil),
	)
}
