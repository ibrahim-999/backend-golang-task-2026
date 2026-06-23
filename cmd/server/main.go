package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/bootstrap"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/observability"
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

	app, err := bootstrap.New(cfg, log, metrics)
	if err != nil {
		return err
	}
	defer app.Stop()

	srv := server.New(cfg.HTTP, app.Router, log)

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
