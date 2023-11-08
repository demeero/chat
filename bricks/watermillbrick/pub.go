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
)

type Publisher struct {
	pub               message.Publisher
	evtPublishCounter metric.Int64Counter
}

func NewPublisher(name string, pub message.Publisher) (*Publisher, error) {
	pub = wotelfloss.NewTracePropagatingPublisherDecorator(pub)
	pub = wotel.NewNamedPublisherDecorator(fmt.Sprintf("%s.publish", name), pub)
	counter, err := otel.GetMeterProvider().Meter("pubsub").Int64Counter("event_publish_count", metric.WithDescription("The number of events published"))
	if err != nil {
		return nil, fmt.Errorf("failed to create event_publish_count metric: %w", err)
	}
	return &Publisher{pub: pub, evtPublishCounter: counter}, nil
}

func (p *Publisher) Publish(topic string, messages ...*message.Message) error {
	err := p.pub.Publish(topic, messages...)
	if len(messages) > 0 {
		p.recordMetrics(messages[0].Context(), topic, err)
	}
	return err
}

func (p *Publisher) Close() error {
	return p.pub.Close()
}

func (p *Publisher) recordMetrics(ctx context.Context, topic string, err error) {
	var resultAttr attribute.KeyValue
	if err != nil {
		resultAttr = semconv.OTelStatusCodeError
	} else {
		resultAttr = semconv.OTelStatusCodeOk
	}
	p.evtPublishCounter.Add(ctx, 1, metric.WithAttributes(semconv.MessagingDestinationName(topic), resultAttr))
}
