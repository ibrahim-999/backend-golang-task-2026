package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/observability"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/handler"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/middleware"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/platform/config"
)

type Dependencies struct {
	Version     string
	Environment string
	Logger      zerolog.Logger
	Metrics     *observability.Metrics
	DB          *gorm.DB

	TokenVerifier ports.TokenVerifier
	RateLimit     config.RateLimit

	Auth    *handler.Auth
	User    *handler.User
	Product *handler.Product
	Order   *handler.Order
	Admin   *handler.Admin
	Docs    *handler.Docs
	WS      gin.HandlerFunc
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
			"docs":    "/docs",
			"api":     "/api/v1",
		})
	})

	if deps.Docs != nil {
		engine.GET("/openapi.yaml", deps.Docs.Spec)
		engine.GET("/docs", deps.Docs.UI)
	}

	if deps.WS != nil {
		engine.GET("/ws", deps.WS)
	}

	v1 := engine.Group("/api/v1")
	if deps.RateLimit.Enabled {
		v1.Use(middleware.RateLimit(deps.RateLimit.RPS, deps.RateLimit.Burst, deps.RateLimit.TTL))
	}
	v1.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"pong": true}) })

	v1.POST("/auth/register", deps.Auth.Register)
	v1.POST("/auth/login", deps.Auth.Login)
	v1.POST("/users", deps.User.Create)
	v1.GET("/products", deps.Product.List)
	v1.GET("/products/:id", deps.Product.Get)

	authed := v1.Group("")
	authed.Use(middleware.Auth(deps.TokenVerifier))

	authed.GET("/users/:id", deps.User.Get)
	authed.PUT("/users/:id", deps.User.Update)

	authed.GET("/products/:id/inventory", deps.Product.Inventory)

	authed.POST("/orders", deps.Order.Place)
	authed.GET("/orders", deps.Order.List)
	authed.GET("/orders/:id", deps.Order.Get)
	authed.PUT("/orders/:id/cancel", deps.Order.Cancel)
	authed.GET("/orders/:id/status", deps.Order.Status)

	admin := v1.Group("")
	admin.Use(middleware.Auth(deps.TokenVerifier), middleware.RequireAdmin())

	admin.POST("/products", deps.Product.Create)
	admin.PUT("/products/:id", deps.Product.Update)

	deps.Admin.Register(admin)

	return engine
}
