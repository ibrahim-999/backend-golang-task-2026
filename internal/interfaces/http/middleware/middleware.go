package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/observability"
)

const (
	RequestIDHeader = "X-Request-ID"
	requestIDKey    = "request_id"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(requestIDKey, id)
		c.Writer.Header().Set(RequestIDHeader, id)
		c.Next()
	}
}

func RequestIDFrom(c *gin.Context) string {
	if v, ok := c.Get(requestIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func Logger(log zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()

		evt := log.Info()
		if c.Writer.Status() >= http.StatusInternalServerError {
			evt = log.Error()
		}
		if len(c.Errors) > 0 {
			evt = evt.Str("gin_errors", c.Errors.String())
		}
		evt.
			Str("request_id", RequestIDFrom(c)).
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", c.Writer.Status()).
			Dur("latency", time.Since(start)).
			Str("client_ip", c.ClientIP()).
			Msg("http_request")
	}
}

func Recovery(log zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error().
					Interface("panic", r).
					Str("request_id", RequestIDFrom(c)).
					Str("path", c.Request.URL.Path).
					Msg("recovered from panic")
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{"code": "internal_error", "message": "internal server error"},
				})
			}
		}()
		c.Next()
	}
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Authorization,Content-Type,"+RequestIDHeader+",Idempotency-Key")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func Metrics(m *observability.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		m.HTTPInFlight.Inc()
		start := time.Now()
		c.Next()
		m.HTTPInFlight.Dec()

		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())
		m.HTTPRequests.WithLabelValues(c.Request.Method, path, status).Inc()
		m.HTTPDuration.WithLabelValues(c.Request.Method, path).Observe(time.Since(start).Seconds())
	}
}
