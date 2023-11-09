package watermillbrick

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	wotelfloss "github.com/dentech-floss/watermill-opentelemetry-go-extra/pkg/opentelemetry"
	wotel "github.com/voi-oss/watermill-opentelemetry/pkg/opentelemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

type PubConfig struct {
	Name                string
	Metrics             bool
	NewRootSpanWithLink bool
}

type Publisher struct {
	cfg               PubConfig
	pub               message.Publisher
	evtPublishCounter metric.Int64Counter
}

func NewPublisher(cfg PubConfig, pub message.Publisher) (*Publisher, error) {
	pub = wotelfloss.NewTracePropagatingPublisherDecorator(pub)
	pub = wotel.NewNamedPublisherDecorator(fmt.Sprintf("%s.publish", cfg.Name), pub)
	counter, err := otel.GetMeterProvider().Meter("bricks/publisher").
		Int64Counter("event_publish_count", metric.WithDescription("The number of events published"))
	if err != nil {
		return nil, fmt.Errorf("failed to create event_publish_count metric: %w", err)
	}
	return &Publisher{pub: pub, evtPublishCounter: counter, cfg: cfg}, nil
}

func (p *Publisher) Publish(topic string, messages ...*message.Message) error {
	if len(messages) == 0 {
		return nil
	}
	ctx := messages[0].Context()
	if p.cfg.NewRootSpanWithLink {
		spanCtx, span := otel.GetTracerProvider().Tracer("bricks/publisher").
			Start(ctx, "publish",
				trace.WithNewRoot(),
				trace.WithLinks(trace.Link{SpanContext: trace.SpanContextFromContext(ctx)}))
		ctx = spanCtx
		defer span.End()
	}

	err := p.pub.Publish(topic, messages...)
	p.recordMetrics(ctx, topic, err)
	return err
}

func (p *Publisher) Close() error {
	return p.pub.Close()
}

func (p *Publisher) recordMetrics(ctx context.Context, topic string, err error) {
	if !p.cfg.Metrics {
		return
	}
	var resultAttr attribute.KeyValue
	if err != nil {
		resultAttr = semconv.OTelStatusCodeError
	} else {
		resultAttr = semconv.OTelStatusCodeOk
	}
	p.evtPublishCounter.Add(ctx, 1, metric.WithAttributes(semconv.MessagingDestinationName(topic), resultAttr))
}
