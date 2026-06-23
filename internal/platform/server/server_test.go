package server_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/platform/config"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/platform/server"
)

func TestServerRunGracefulShutdown(t *testing.T) {
	srv := server.New(config.HTTP{
		Host:            "127.0.0.1",
		Port:            0,
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		IdleTimeout:     time.Second,
		ShutdownTimeout: 2 * time.Second,
	}, http.NotFoundHandler(), zerolog.Nop())

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	require.NoError(t, srv.Run(ctx))
}
