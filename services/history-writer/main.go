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
	"github.com/demeero/chat/bricks/cassandra"
	"github.com/demeero/chat/bricks/logger"
	"github.com/demeero/chat/bricks/meter"
	"github.com/demeero/chat/bricks/tracer"
	"github.com/demeero/chat/bricks/watermillbrick"
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
	logger.Configure(logger.Config{
		Level:     cfg.Log.Level,
		AddSource: cfg.Log.AddSource,
		JSON:      cfg.Log.JSON,
	})

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Addr,
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
	defer cSess.Close()

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

	wmLogger := watermill.NewSlogLogger(slog.Default())
	publisher, err := redisstream.NewPublisher(redisstream.PublisherConfig{Client: rdb}, wmLogger)
	if err != nil {
		log.Fatalf("failed create redisstream publisher: %s", err)
	}
	pub, err := watermillbrick.NewPublisher(watermillbrick.PubConfig{
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
	if err := tracerShutdown(context.Background()); err != nil {
		slog.Error("failed shutdown tracer provider", slog.Any("err", err))
	}
}
