package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var (
	Logger   = gin.Logger
	Recovery = gin.Recovery
	CORS     = cors.New
)

type CORSConfig = cors.Config
