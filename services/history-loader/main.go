package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/demeero/chat/bricks/cassandra"
	"github.com/demeero/chat/bricks/logger"
	"github.com/demeero/chat/bricks/meter"
	"github.com/demeero/chat/bricks/tracer"
	"github.com/gocql/gocql"
	_ "go.uber.org/automaxprocs"
)

var roomChatID gocql.UUID

func main() {
	uuid, err := gocql.ParseUUID("2f3025ab-9cf7-48a8-9f61-e0f5924ec6d4")
	if err != nil {
		panic(err)
	}
	roomChatID = uuid

	cfg := LoadConfig()
	logger.Configure(logger.Config{
		Level:     cfg.Log.Level,
		AddSource: cfg.Log.AddSource,
		JSON:      cfg.Log.JSON,
	})

	cluster := gocql.NewCluster(cfg.Cassandra.Host)
	cluster.Keyspace = cfg.Cassandra.Keyspace
	cluster.QueryObserver = cassandra.NewObserverChain(
		cassandra.LogQueryObserver{Disabled: !cfg.Cassandra.Log},
		cassandra.TraceQueryObserver{},
		cassandra.MeterQueryObserver{})
	cSess, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("failed create cassandra session: %s", err)
	}

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer cancel()

	tracerShutdown, err := tracer.Init(ctx, tracer.Config{
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

	loader := Loader{Sess: cSess}

	httpSrv := setupHTTPSrv(ctx, cfg, loader)

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, shutdownCtxCancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer shutdownCtxCancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed shutdown http srv", slog.Any("err", err))
	}
	if err := meterShutdown(context.Background()); err != nil {
		slog.Error("failed shutdown meter provider", slog.Any("err", err))
	}
	if err := tracerShutdown(context.Background()); err != nil {
		slog.Error("failed shutdown tracer provider", slog.Any("err", err))
	}
	cSess.Close()
}
