package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/platform/config"
)

type Server struct {
	httpServer      *http.Server
	shutdownTimeout time.Duration
	log             zerolog.Logger
}

func New(cfg config.HTTP, handler http.Handler, log zerolog.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         cfg.Addr(),
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
		shutdownTimeout: cfg.ShutdownTimeout,
		log:             log,
	}
}

func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.log.Info().Str("addr", s.httpServer.Addr).Msg("http server listening")
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.log.Info().Msg("shutdown signal received, draining http server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		s.log.Info().Msg("http server stopped cleanly")
		return nil
	}
}
