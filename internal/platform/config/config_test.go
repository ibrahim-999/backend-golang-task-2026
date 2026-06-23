package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/platform/config"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	assert.NotEmpty(t, cfg.App.Name)
	assert.False(t, cfg.App.IsProduction())
	assert.Contains(t, cfg.HTTP.Addr(), ":")
	assert.Contains(t, cfg.DB.DSN(), "dbname=")
	assert.Contains(t, cfg.RabbitMQ.URL(), "amqp://")
	assert.NotEmpty(t, cfg.JWT.Secret)
	assert.GreaterOrEqual(t, cfg.Worker.OrderWorkers, 1)
}

func TestLoadOverridesFromEnv(t *testing.T) {
	t.Setenv("HTTP_PORT", "9999")
	t.Setenv("PAYMENT_FAILURE_RATE", "0.25")
	t.Setenv("RATE_LIMIT_RPS", "33")
	t.Setenv("LOG_PRETTY", "false")
	t.Setenv("HTTP_READ_TIMEOUT", "5s")
	t.Setenv("ORDER_WORKERS", "12")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, 9999, cfg.HTTP.Port)
	assert.InDelta(t, 0.25, cfg.Payment.FailureRate, 0.0001)
	assert.InDelta(t, 33.0, cfg.RateLimit.RPS, 0.0001)
	assert.False(t, cfg.App.LogPretty)
	assert.Equal(t, 5*time.Second, cfg.HTTP.ReadTimeout)
	assert.Equal(t, 12, cfg.Worker.OrderWorkers)
}

func TestLoadProductionRejectsWeakSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "short")

	_, err := config.Load()
	require.Error(t, err)
}

func TestLoadRejectsInvalidFailureRate(t *testing.T) {
	t.Setenv("PAYMENT_FAILURE_RATE", "2.0")

	_, err := config.Load()
	require.Error(t, err)
}

func TestLoadRejectsZeroWorkers(t *testing.T) {
	t.Setenv("ORDER_WORKERS", "0")

	_, err := config.Load()
	require.Error(t, err)
}

func TestLoadGeneratesDevSecretWhenEmpty(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWT_SECRET", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(cfg.JWT.Secret), 32)
}
