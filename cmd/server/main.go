package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/observability"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/persistence/gormrepo"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/router"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/platform/config"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/platform/server"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := observability.NewLogger(cfg.App.Name, cfg.App.LogLevel, cfg.App.LogPretty)
	metrics := observability.NewMetrics("orders")

	db, err := gormrepo.Open(cfg.DB, log)
	if err != nil {
		return err
	}
	defer func() {
		if sqlDB, derr := db.DB(); derr == nil {
			_ = sqlDB.Close()
		}
	}()

	engine := router.New(router.Dependencies{
		Version:     cfg.App.Version,
		Environment: cfg.App.Environment,
		Logger:      log,
		Metrics:     metrics,
		DB:          db,
	})

	srv := server.New(cfg.HTTP, engine, log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info().
		Str("version", cfg.App.Version).
		Str("environment", cfg.App.Environment).
		Str("addr", cfg.HTTP.Addr()).
		Msg("starting order-processing service")

	return srv.Run(ctx)
}

func healthcheck() int {
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%s/health", port))
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
