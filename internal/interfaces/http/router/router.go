package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/observability"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/handler"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/middleware"
)

type Dependencies struct {
	Version     string
	Environment string
	Logger      zerolog.Logger
	Metrics     *observability.Metrics
	DB          *gorm.DB
}

func New(deps Dependencies) *gin.Engine {
	if deps.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(
		middleware.RequestID(),
		middleware.Recovery(deps.Logger),
		middleware.Logger(deps.Logger),
		middleware.Metrics(deps.Metrics),
		middleware.CORS(),
	)

	health := handler.NewHealth(deps.Version, deps.DB)
	engine.GET("/health", health.Live)
	engine.GET("/ready", health.Ready)
	engine.GET("/metrics", gin.WrapH(deps.Metrics.Handler()))

	engine.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "order-processing",
			"version": deps.Version,
			"health":  "/health",
			"ready":   "/ready",
			"metrics": "/metrics",
			"api":     "/api/v1",
		})
	})

	v1 := engine.Group("/api/v1")
	v1.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"pong": true}) })

	return engine
}
