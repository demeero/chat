package logger

import (
	"context"
	"log/slog"
	"os"
)

type logCtxKey struct{}

var logKey = logCtxKey{}

type Config struct {
	Level     string
	AddSource bool
	JSON      bool
}

func Configure(cfg Config) {
	logLvl := &slog.LevelVar{}
	if err := logLvl.UnmarshalText([]byte(cfg.Level)); err != nil {
		slog.Error("failed parse log level - use info",
			slog.Any("err", err), slog.String("level", cfg.Level))
		logLvl.Set(slog.LevelInfo)
	}
	opts := &slog.HandlerOptions{
		Level:     logLvl,
		AddSource: cfg.AddSource,
	}
	var h slog.Handler
	if cfg.JSON {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	logger := slog.New(h)
	slog.SetDefault(logger)
	slog.Info("log configured")
}

func FromCtx(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(logKey).(*slog.Logger)
	if !ok {
		slog.Debug("no logger in context, using default")
		return slog.Default()
	}
	return logger
}

func ToCtx(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, logKey, logger)
}
