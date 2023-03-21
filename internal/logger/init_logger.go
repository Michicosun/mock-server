package logger

import (
	"io"
	"mock-server/internal/configs"
	"os"
	"path"
	"time"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

func newRollingFile(cfg *configs.LogConfig) io.Writer {
	if err := os.MkdirAll(cfg.Directory, 0744); err != nil {
		zlog.Error().Err(err).Str("path", cfg.Directory).Msg("failed to create log directory")
		return nil
	}

	return &lumberjack.Logger{
		Filename:   path.Join(cfg.Directory, cfg.Filename),
		MaxBackups: cfg.MaxBackups, // files
		MaxSize:    cfg.MaxSize,    // megabytes
		MaxAge:     cfg.MaxAge,     // days
	}
}

func Init(cfg *configs.LogConfig) {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	writers := make([]io.Writer, 0)

	if cfg.ConsoleLoggingEnabled {
		writers = append(writers, zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "Jan _2 15:04:05.00000",
		})
	}

	if cfg.FileLoggingEnabled {
		file_logger := newRollingFile(cfg)
		if file_logger != nil {
			writers = append(writers, file_logger)
		}
	}

	mw := io.MultiWriter(writers...)

	zlog.Logger = zerolog.
		New(mw).
		Level(zerolog.Level(cfg.Level)).
		With().
		Timestamp().
		Caller().
		Logger()
}
