package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App       App
	HTTP      HTTP
	DB        DB
	JWT       JWT
	RateLimit RateLimit
	Worker    Worker
	Payment   Payment
	RabbitMQ  RabbitMQ
}

type App struct {
	Name        string
	Environment string
	Version     string
	LogLevel    string
	LogPretty   bool
}

func (a App) IsProduction() bool { return strings.EqualFold(a.Environment, "production") }

type HTTP struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

func (h HTTP) Addr() string { return fmt.Sprintf("%s:%d", h.Host, h.Port) }

type DB struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	ConnectRetries  int
	ConnectBackoff  time.Duration
}

func (d DB) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

type JWT struct {
	Secret    string
	Issuer    string
	AccessTTL time.Duration
}

type RateLimit struct {
	Enabled bool
	RPS     float64
	Burst   int
	TTL     time.Duration
}

type Worker struct {
	OrderWorkers          int
	OrderQueueSize        int
	NotificationWorkers   int
	NotificationQueueSize int
	EventWorkers          int
	EventQueueSize        int
}

type Payment struct {
	Provider     string
	FailureRate  float64
	MinLatency   time.Duration
	MaxLatency   time.Duration
	MaxAttempts  int
	RetryBackoff time.Duration
}

type RabbitMQ struct {
	Enabled  bool
	Host     string
	Port     int
	User     string
	Password string
	VHost    string
	Exchange string
}

func (r RabbitMQ) URL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/%s", r.User, r.Password, r.Host, r.Port, strings.TrimPrefix(r.VHost, "/"))
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		App: App{
			Name:        getString("APP_NAME", "order-processing"),
			Environment: getString("APP_ENV", "development"),
			Version:     getString("APP_VERSION", "dev"),
			LogLevel:    getString("LOG_LEVEL", "info"),
			LogPretty:   getBool("LOG_PRETTY", true),
		},
		HTTP: HTTP{
			Host:            getString("HTTP_HOST", "0.0.0.0"),
			Port:            getInt("HTTP_PORT", 8080),
			ReadTimeout:     getDuration("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     getDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getDuration("HTTP_SHUTDOWN_TIMEOUT", 20*time.Second),
		},
		DB: DB{
			Host:            getString("DB_HOST", "localhost"),
			Port:            getInt("DB_PORT", 5432),
			User:            getString("DB_USER", "postgres"),
			Password:        getString("DB_PASSWORD", ""),
			Name:            getString("DB_NAME", "orders"),
			SSLMode:         getString("DB_SSLMODE", "disable"),
			MaxOpenConns:    getInt("DB_MAX_OPEN_CONNS", 50),
			MaxIdleConns:    getInt("DB_MAX_IDLE_CONNS", 25),
			ConnMaxLifetime: getDuration("DB_CONN_MAX_LIFETIME", time.Hour),
			ConnMaxIdleTime: getDuration("DB_CONN_MAX_IDLE_TIME", 30*time.Minute),
			ConnectRetries:  getInt("DB_CONNECT_RETRIES", 10),
			ConnectBackoff:  getDuration("DB_CONNECT_BACKOFF", 2*time.Second),
		},
		JWT: JWT{
			Secret:    getString("JWT_SECRET", ""),
			Issuer:    getString("JWT_ISSUER", "order-processing"),
			AccessTTL: getDuration("JWT_ACCESS_TTL", 24*time.Hour),
		},
		RateLimit: RateLimit{
			Enabled: getBool("RATE_LIMIT_ENABLED", true),
			RPS:     getFloat("RATE_LIMIT_RPS", 20),
			Burst:   getInt("RATE_LIMIT_BURST", 40),
			TTL:     getDuration("RATE_LIMIT_TTL", 10*time.Minute),
		},
		Worker: Worker{
			OrderWorkers:          getInt("ORDER_WORKERS", 8),
			OrderQueueSize:        getInt("ORDER_QUEUE_SIZE", 1024),
			NotificationWorkers:   getInt("NOTIFICATION_WORKERS", 4),
			NotificationQueueSize: getInt("NOTIFICATION_QUEUE_SIZE", 2048),
			EventWorkers:          getInt("EVENT_WORKERS", 4),
			EventQueueSize:        getInt("EVENT_QUEUE_SIZE", 2048),
		},
		Payment: Payment{
			Provider:     getString("PAYMENT_PROVIDER", "mock"),
			FailureRate:  getFloat("PAYMENT_FAILURE_RATE", 0.1),
			MinLatency:   getDuration("PAYMENT_MIN_LATENCY", 20*time.Millisecond),
			MaxLatency:   getDuration("PAYMENT_MAX_LATENCY", 120*time.Millisecond),
			MaxAttempts:  getInt("PAYMENT_MAX_ATTEMPTS", 3),
			RetryBackoff: getDuration("PAYMENT_RETRY_BACKOFF", 50*time.Millisecond),
		},
		RabbitMQ: RabbitMQ{
			Enabled:  getBool("RABBITMQ_ENABLED", false),
			Host:     getString("RABBITMQ_HOST", "rabbitmq"),
			Port:     getInt("RABBITMQ_PORT", 5672),
			User:     getString("RABBITMQ_USER", "guest"),
			Password: getString("RABBITMQ_PASSWORD", ""),
			VHost:    getString("RABBITMQ_VHOST", "/"),
			Exchange: getString("RABBITMQ_EXCHANGE", "orders.events"),
		},
	}

	if cfg.JWT.Secret == "" && !cfg.App.IsProduction() {
		cfg.JWT.Secret = randomSecret(32)
	}

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	if c.JWT.Secret == "" {
		return errors.New("JWT_SECRET is required (provide it via the environment)")
	}
	if c.App.IsProduction() && len(c.JWT.Secret) < 32 {
		return errors.New("JWT_SECRET must be at least 32 bytes in production")
	}
	if c.Payment.FailureRate < 0 || c.Payment.FailureRate > 1 {
		return errors.New("PAYMENT_FAILURE_RATE must be between 0 and 1")
	}
	if c.Worker.OrderWorkers < 1 {
		return errors.New("ORDER_WORKERS must be at least 1")
	}
	return nil
}

func randomSecret(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func getString(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getFloat(key string, fallback float64) float64 {
	if v, ok := os.LookupEnv(key); ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
