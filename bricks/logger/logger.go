package logger

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

type logCtxKey struct{}

var logKey = logCtxKey{}

type Config struct {
	Level     string
	AddSource bool
	JSON      bool
	Fields    map[string]string
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
	if len(cfg.Fields) > 0 {
		for k, v := range cfg.Fields {
			logger = logger.With(slog.String(k, v))
		}
	}
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

func InstrumentWithTrace(ctx context.Context, logger *slog.Logger) *slog.Logger {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasSpanID() {
		logger = logger.With(slog.String("otel_span_id", spanCtx.SpanID().String()))
	}
	if spanCtx.HasTraceID() {
		logger = logger.With(slog.String("otel_trace_id", spanCtx.TraceID().String()))
	}
	return logger
}
