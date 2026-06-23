package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/observability"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/httpx"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

const (
	RequestIDHeader = "X-Request-ID"
	requestIDKey    = "request_id"
	userIDKey       = "user_id"
	userRoleKey     = "user_role"
	roleAdmin       = "admin"
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

func Auth(verifier ports.TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
			httpx.Error(c, errs.Unauthorized("unauthorized", "missing or malformed authorization header"))
			return
		}

		token := strings.TrimSpace(header[len(prefix):])
		claims, err := verifier.Verify(token)
		if err != nil {
			httpx.Error(c, errs.Unauthorized("unauthorized", "invalid or expired token"))
			return
		}

		c.Set(userIDKey, claims.UserID)
		c.Set(userRoleKey, claims.Role)
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if RoleFrom(c) != roleAdmin {
			httpx.Error(c, errs.Forbidden("forbidden", "admin role required"))
			return
		}
		c.Next()
	}
}

func UserIDFrom(c *gin.Context) (uint64, bool) {
	if v, ok := c.Get(userIDKey); ok {
		if id, ok := v.(uint64); ok {
			return id, true
		}
	}
	return 0, false
}

func RoleFrom(c *gin.Context) string {
	if v, ok := c.Get(userRoleKey); ok {
		if role, ok := v.(string); ok {
			return role
		}
	}
	return ""
}

type rateEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func RateLimit(rps float64, burst int, ttl time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	entries := make(map[string]*rateEntry)
	lastEvict := time.Now()

	return func(c *gin.Context) {
		client := c.ClientIP()
		now := time.Now()

		mu.Lock()
		if now.Sub(lastEvict) >= ttl {
			for key, e := range entries {
				if now.Sub(e.lastSeen) >= ttl {
					delete(entries, key)
				}
			}
			lastEvict = now
		}

		e, ok := entries[client]
		if !ok {
			e = &rateEntry{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
			entries[client] = e
		}
		e.lastSeen = now
		allowed := e.limiter.Allow()
		mu.Unlock()

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{"code": "rate_limited", "message": "too many requests"},
			})
			return
		}
		c.Next()
	}
}
