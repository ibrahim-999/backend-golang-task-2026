package observability

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

func NewLogger(service, level string, pretty bool) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	parsed, err := zerolog.ParseLevel(level)
	if err != nil {
		parsed = zerolog.InfoLevel
	}
	var w io.Writer = os.Stdout
	if pretty {
		w = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}
	return zerolog.New(w).Level(parsed).With().Timestamp().Str("service", service).Logger()
}
