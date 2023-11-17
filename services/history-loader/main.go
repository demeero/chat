package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/demeero/bricks/configbrick"
	"github.com/demeero/bricks/cqlbrick"
	"github.com/demeero/bricks/otelbrick"
	"github.com/demeero/bricks/slogbrick"
	"github.com/gocql/gocql"
	_ "go.uber.org/automaxprocs"
)

type Config struct {
	configbrick.AppMeta
	Cassandra configbrick.Cassandra `json:"cassandra"`
	Log       configbrick.Log       `json:"log"`
	HTTP      configbrick.HTTP      `json:"http"`
	JwksURL   string                `split_words:"true" json:"jwks_url"`
	OTEL      configbrick.OTEL      `json:"otel"`
}

var roomChatID gocql.UUID

func main() {
	uuid, err := gocql.ParseUUID("2f3025ab-9cf7-48a8-9f61-e0f5924ec6d4")
	if err != nil {
		panic(err)
	}
	roomChatID = uuid

	cfg := Config{}
	configbrick.LoadConfig(&cfg, os.Getenv("LOG_CONFIG") == "true")

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

	traceCfg := cfg.OTEL.Trace
	traceShutdown, err := otelbrick.InitTrace(ctx, otelbrick.TraceConfig{
		ServiceName:           cfg.ServiceName,
		ServiceNamespace:      cfg.ServiceNamespace,
		DeploymentEnvironment: cfg.Env,
		OTELHTTPEndpoint:      traceCfg.Endpoint,
		OTELHTTPPathPrefix:    traceCfg.PathPrefix,
		Insecure:              traceCfg.Insecure,
		Headers:               traceCfg.BasicAuthHeader(),
	})
	if err != nil {
		log.Fatalf("failed init tracer: %s", err)
	}

	meterCfg := cfg.OTEL.Meter
	meterShutdown, err := otelbrick.InitMeter(ctx, otelbrick.MeterConfig{
		ServiceName:           cfg.ServiceName,
		ServiceNamespace:      cfg.ServiceNamespace,
		DeploymentEnvironment: cfg.Env,
		OTELHTTPEndpoint:      meterCfg.Endpoint,
		OTELHTTPPathPrefix:    meterCfg.PathPrefix,
		Insecure:              meterCfg.Insecure,
		RuntimeMetrics:        true,
		HostMetrics:           true,
		Headers:               meterCfg.BasicAuthHeader(),
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
