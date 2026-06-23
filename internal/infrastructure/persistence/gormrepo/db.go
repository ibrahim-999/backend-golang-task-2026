package gormrepo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/platform/config"
)

func Open(cfg config.DB, log zerolog.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger:                 gormlogger.Default.LogMode(gormlogger.Warn),
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		return nil, fmt.Errorf("open gorm: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("resolve sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	if err := waitForDB(sqlDB, cfg, log); err != nil {
		return nil, err
	}

	log.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.Name).
		Int("max_open_conns", cfg.MaxOpenConns).
		Msg("database connected")
	return db, nil
}

func waitForDB(sqlDB *sql.DB, cfg config.DB, log zerolog.Logger) error {
	var lastErr error
	for attempt := 1; attempt <= cfg.ConnectRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := sqlDB.PingContext(ctx)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		log.Warn().
			Err(err).
			Int("attempt", attempt).
			Int("max_attempts", cfg.ConnectRetries).
			Msg("database not ready, retrying")
		time.Sleep(cfg.ConnectBackoff)
	}
	return fmt.Errorf("database unreachable after %d attempts: %w", cfg.ConnectRetries, lastErr)
}
