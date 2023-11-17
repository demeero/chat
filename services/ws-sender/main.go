package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/demeero/bricks/configbrick"
	"github.com/demeero/bricks/otelbrick"
	"github.com/demeero/bricks/slogbrick"
	"github.com/demeero/bricks/watermillbrick"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	_ "go.uber.org/automaxprocs"
)

type Config struct {
	configbrick.AppMeta
	Redis   configbrick.Redis `json:"redis"`
	Log     configbrick.Log   `json:"log"`
	HTTP    configbrick.HTTP  `json:"http"`
	JwksURL string            `split_words:"true" json:"jwks_url"`
	OTEL    configbrick.OTEL  `json:"otel"`
}

func main() {
	cfg := Config{}
	configbrick.LoadConfig(&cfg, os.Getenv("LOG_CONFIG") == "true")
	slogbrick.Configure(slogbrick.Config{
		Level:     cfg.Log.Level,
		AddSource: cfg.Log.AddSource,
		JSON:      cfg.Log.JSON,
	})

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

	rdb := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB})
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
