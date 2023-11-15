package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/demeero/bricks/cqlbrick"
	"github.com/demeero/bricks/otelbrick"
	"github.com/demeero/bricks/slogbrick"
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
	slogbrick.Configure(slogbrick.Config{
		Level:     cfg.Log.Level,
		AddSource: cfg.Log.AddSource,
		JSON:      cfg.Log.JSON,
	})

	cluster := gocql.NewCluster(cfg.Cassandra.Host)
	cluster.Keyspace = cfg.Cassandra.Keyspace
	cqlMeterObsrvr, err := cqlbrick.NewOTELMeterQueryObserver(false)
	if err != nil {
		log.Fatalf("failed create cql meter observer: %s", err)
	}
	cluster.QueryObserver = cqlbrick.NewObserverChain(
		cqlbrick.SlogLogQueryObserver{Disabled: !cfg.Cassandra.Log},
		cqlbrick.NewOTELTraceQueryObserver(false),
		cqlMeterObsrvr)
	cSess, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("failed create cassandra session: %s", err)
	}

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
		OTELHTTPPathPrefix:    cfg.Telemetry.PathPrefix,
		Insecure:              true,
		RuntimeMetrics:        true,
		HostMetrics:           true,
		Headers:               cfg.Telemetry.TraceBasicAuthHeader(),
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
	if err := traceShutdown(context.Background()); err != nil {
		slog.Error("failed shutdown tracer provider", slog.Any("err", err))
	}
	cSess.Close()
}
