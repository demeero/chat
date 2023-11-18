package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"github.com/demeero/bricks/configbrick"
	"github.com/demeero/bricks/cqlbrick"
	"github.com/demeero/bricks/otelbrick"
	"github.com/demeero/bricks/slogbrick"
	"github.com/demeero/chat/bricks/httpsrv"
	"github.com/demeero/chat/history/httphandler"
	"github.com/demeero/chat/history/loader"
	"github.com/gocql/gocql"
)

type config struct {
	configbrick.AppMeta
	Cassandra configbrick.Cassandra `json:"cassandra"`
	Log       configbrick.Log       `json:"log"`
	HTTP      configbrick.HTTP      `json:"http"`
	JwksURL   string                `split_words:"true" json:"jwks_url"`
	OTEL      configbrick.OTEL      `json:"otel"`
}

func main() {
	cfg := config{}
	configbrick.LoadConfig(&cfg, os.Getenv("LOG_CONFIG") == "true")

	slogbrick.Configure(slogbrick.Config{
		Level:     cfg.Log.Level,
		AddSource: cfg.Log.AddSource,
		JSON:      cfg.Log.JSON,
	})

	cluster := gocql.NewCluster(cfg.Cassandra.Host)
	if cfg.Cassandra.Username != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: cfg.Cassandra.Username,
			Password: cfg.Cassandra.Password,
		}
	}
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

	httpCfg := cfg.HTTP
	httpSrv := httpsrv.Configure(httpsrv.Config{
		ReadHeaderTimeout: httpCfg.ReadHeaderTimeout,
		ReadTimeout:       httpCfg.ReadTimeout,
		WriteTimeout:      httpCfg.WriteTimeout,
		Port:              httpCfg.Port,
	})
	if err := httphandler.Setup(ctx, cfg.JwksURL, cfg.ServiceName, httpSrv, loader.New(cSess)); err != nil {
		log.Fatalf("failed setup http handler: %s", err)
	}
	go func() {
		slog.Info("initializing HTTP server", slog.Int("port", httpCfg.Port))
		if err := httpSrv.Start(fmt.Sprintf(":%d", httpCfg.Port)); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed http server: %s", err)
		}
	}()

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
