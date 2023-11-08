package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/demeero/chat/bricks/logger"
	"github.com/demeero/chat/bricks/meter"
	"github.com/demeero/chat/bricks/tracer"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	wotel "github.com/voi-oss/watermill-opentelemetry/pkg/opentelemetry"
)

func main() {
	cfg := LoadConfig()
	logger.Configure(logger.Config{
		Level:     cfg.Log.Level,
		AddSource: cfg.Log.AddSource,
		JSON:      cfg.Log.JSON,
	})

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer cancel()

	traceShutdown, err := tracer.Init(ctx, tracer.Config{
		ServiceName:           cfg.ServiceName,
		ServiceNamespace:      cfg.ServiceNamespace,
		DeploymentEnvironment: cfg.Env,
		OTELEndpoint:          cfg.Telemetry.GrpcOtelEndpoint,
		Insecure:              true,
	})
	if err != nil {
		log.Fatalf("failed init tracer: %s", err)
	}

	meterShutdown, err := meter.Init(ctx, meter.Config{
		ServiceName:           cfg.ServiceName,
		ServiceNamespace:      cfg.ServiceNamespace,
		DeploymentEnvironment: cfg.Env,
		OTELEndpoint:          cfg.Telemetry.HttpOtelEndpoint,
		Insecure:              true,
		RuntimeMetrics:        true,
		HostMetrics:           true,
	})
	if err != nil {
		log.Fatalf("failed init metrics: %s", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr})
	if err := redisotel.InstrumentTracing(rdb, redisotel.WithDBStatement(true)); err != nil {
		log.Fatalf("failed instrument redis with tracing: %s", err)
	}
	if err := redisotel.InstrumentMetrics(rdb); err != nil {
		log.Fatalf("failed instrument redis with metrics: %s", err)
	}

	pub, err := redisstream.NewPublisher(redisstream.PublisherConfig{Client: rdb}, watermill.NewSlogLogger(slog.Default()))
	if err != nil {
		log.Fatalf("failed create redisstream publisher: %s", err)
	}
	publisher := wotel.NewNamedPublisherDecorator("ws-sender", pub)

	httpSrv := setupHTTPSrv(ctx, cfg, publisher)

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, shutdownCtxCancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer shutdownCtxCancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed shutdown http srv", slog.Any("err", err))
	}
	if err := pub.Close(); err != nil {
		slog.Error("failed close redisstream publisher", slog.Any("err", err))
	}
	if err := rdb.Close(); err != nil {
		slog.Error("failed close redis conn", slog.Any("err", err))
	}
	if err := meterShutdown(context.Background()); err != nil {
		slog.Error("failed shutdown meter provider", slog.Any("err", err))
	}
	if err := traceShutdown(context.Background()); err != nil {
		slog.Error("error shutdown tracer provider", slog.Any("err", err))
	}
}
