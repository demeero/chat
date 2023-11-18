package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/demeero/bricks/configbrick"
	"github.com/demeero/bricks/cqlbrick"
	"github.com/demeero/bricks/otelbrick"
	"github.com/demeero/bricks/slogbrick"
	"github.com/demeero/bricks/watermillbrick"
	"github.com/demeero/chat/history/event"
	"github.com/demeero/chat/history/writer"
	wotelfloss "github.com/dentech-floss/watermill-opentelemetry-go-extra/pkg/opentelemetry"
	"github.com/gocql/gocql"
	"github.com/redis/go-redis/v9"
	wotel "github.com/voi-oss/watermill-opentelemetry/pkg/opentelemetry"
)

const topic = "msg_sent"

type config struct {
	configbrick.AppMeta
	Redis     configbrick.Redis     `json:"redis"`
	Cassandra configbrick.Cassandra `json:"cassandra"`
	Log       configbrick.Log       `json:"log"`
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
		log.Fatalf("failed create scylladb session: %s", err)
	}
	defer cSess.Close()

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
	wmLogger := watermill.NewSlogLogger(slog.Default())
	publisher, err := redisstream.NewPublisher(redisstream.PublisherConfig{Client: rdb}, wmLogger)
	if err != nil {
		log.Fatalf("failed create redisstream publisher: %s", err)
	}
	pub, err := watermillbrick.NewOTELPublisher(watermillbrick.OTELPubConfig{
		Name:    "history-writer",
		Metrics: true,
	}, publisher)
	if err != nil {
		log.Fatalf("failed create watermill publisher: %s", err)
	}
	sub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        rdb,
		ConsumerGroup: "history-writer",
	}, wmLogger)
	if err != nil {
		log.Fatalf("failed create redisstream subscriber: %s", err)
	}

	r, err := message.NewRouter(message.RouterConfig{}, wmLogger)
	if err != nil {
		log.Fatalf("failed create watermill router: %s", err)
	}
	r.AddMiddleware(wotelfloss.ExtractRemoteParentSpanContext())
	r.AddMiddleware(wotel.Trace())
	r.AddHandler("history-writer",
		topic,
		sub,
		"msg_stored",
		pub,
		event.MsgSentEvtHandler(topic, writer.New(cSess)))
	go func() {
		if err := r.Run(ctx); err != nil {
			log.Fatalf("failed run watermill router: %s", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	if err := r.Close(); err != nil {
		slog.Error("failed close watermill router", slog.Any("err", err))
	}
	if err := publisher.Close(); err != nil {
		slog.Error("failed close redisstream publisher", slog.Any("err", err))
	}
	if err := rdb.Close(); err != nil {
		slog.Error("failed close redisstream publisher", slog.Any("err", err))
	}
	if err := meterShutdown(context.Background()); err != nil {
		slog.Error("failed shutdown meter provider", slog.Any("err", err))
	}
	if err := traceShutdown(context.Background()); err != nil {
		slog.Error("failed shutdown tracer provider", slog.Any("err", err))
	}
}
