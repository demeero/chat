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
	"github.com/demeero/bricks/cqlbrick"
	"github.com/demeero/bricks/otelbrick"
	"github.com/demeero/bricks/slogbrick"
	"github.com/demeero/bricks/watermillbrick"
	wotelfloss "github.com/dentech-floss/watermill-opentelemetry-go-extra/pkg/opentelemetry"
	"github.com/gocql/gocql"
	"github.com/redis/go-redis/v9"
	wotel "github.com/voi-oss/watermill-opentelemetry/pkg/opentelemetry"

	_ "go.uber.org/automaxprocs"
)

const topic = "msg_sent"

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

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Addr,
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

	traceShutdown, err := otelbrick.InitTrace(ctx, otelbrick.TraceConfig{
		ServiceName:           cfg.ServiceName,
		ServiceNamespace:      cfg.ServiceNamespace,
		DeploymentEnvironment: cfg.Env,
		OTELHTTPEndpoint:      cfg.Telemetry.TraceEndpoint,
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
		msgEvtHandler(cSess))
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
