package cassandra

import (
	"context"
	"log/slog"

	"github.com/demeero/chat/bricks/logger"
	"github.com/gocql/gocql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type QueryObserverChain struct {
	observers []gocql.QueryObserver
}

func NewObserverChain(observers ...gocql.QueryObserver) QueryObserverChain {
	return QueryObserverChain{observers: observers}
}

func (o QueryObserverChain) ObserveQuery(ctx context.Context, q gocql.ObservedQuery) {
	for _, o := range o.observers {
		o.ObserveQuery(ctx, q)
	}
}

type LogQueryObserver struct {
	Disabled bool
}

func (o LogQueryObserver) ObserveQuery(ctx context.Context, q gocql.ObservedQuery) {
	if o.Disabled {
		return
	}
	lg := logger.FromCtx(ctx).With(slog.Int64("latency", q.Metrics.TotalLatency/1e6),
		slog.String("statement", q.Statement),
		slog.String("keyspace", q.Keyspace),
		slog.Int("rows", q.Rows),
		slog.Int("attempt", q.Attempt))
	if q.Err != nil {
		lg = lg.With(slog.Any("err", q.Err))
	}
	lg.Debug("cql query")
}

type TraceQueryObserver struct {
	Disabled bool
}

func (o TraceQueryObserver) ObserveQuery(ctx context.Context, q gocql.ObservedQuery) {
	if o.Disabled {
		return
	}
	t := otel.GetTracerProvider().Tracer("cassandra")
	ctx, span := t.Start(ctx, "cql-query", trace.WithTimestamp(q.Start.UTC()))
	span.SetAttributes(attribute.String("statement", q.Statement),
		attribute.String("keyspace", q.Keyspace),
		attribute.Int("rows", q.Rows),
		attribute.Int("attempt", q.Attempt))
	if q.Err != nil {
		span.RecordError(q.Err)
		span.SetStatus(codes.Error, "")
	}
	span.SetStatus(codes.Ok, "")
	span.End(trace.WithTimestamp(q.End.UTC()))
}

type MeterQueryObserver struct {
	Disabled bool
}

func (o MeterQueryObserver) ObserveQuery(ctx context.Context, q gocql.ObservedQuery) {
	if o.Disabled {
		return
	}
	meter := otel.GetMeterProvider().Meter("cassandra")
	_ = meter
	// TODO
}
