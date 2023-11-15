package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/demeero/bricks/otelbrick"
	"github.com/demeero/bricks/slogbrick"
	"github.com/demeero/bricks/watermillbrick"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	_ "go.uber.org/automaxprocs"
)

func main() {
	cfg := LoadConfig()
	slogbrick.Configure(slogbrick.Config{
		Level:     cfg.Log.Level,
		AddSource: cfg.Log.AddSource,
		JSON:      cfg.Log.JSON,
	})

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer cancel()

	traceShutdown, err := otelbrick.InitTrace(ctx, otelbrick.TraceConfig{
		ServiceName:           cfg.ServiceName,
		ServiceNamespace:      cfg.ServiceNamespace,
		DeploymentEnvironment: cfg.Env,
		OTELHTTPEndpoint:      cfg.Telemetry.TraceEndpoint,
		OTELHTTPPathPrefix:    cfg.Telemetry.PathPrefix,
		Insecure:              true,
		Headers:               cfg.Telemetry.TraceBasicAuthHeader(),
	})
	if err != nil {
		log.Fatalf("failed init tracer: %s", err)
	}

	meterShutdown, err := otelbrick.InitMeter(ctx, otelbrick.MeterConfig{
		ServiceName:           cfg.ServiceName,
		ServiceNamespace:      cfg.ServiceNamespace,
		DeploymentEnvironment: cfg.Env,
		OTELHTTPEndpoint:      cfg.Telemetry.MeterEndpoint,
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
	publisher, err := watermillbrick.NewOTELPublisher(watermillbrick.OTELPubConfig{
		Name:                "ws-sender",
		Metrics:             true,
		NewRootSpanWithLink: true,
	}, pub)
	if err != nil {
		log.Fatalf("failed create instrumented watermill publisher: %s", err)
	}

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
