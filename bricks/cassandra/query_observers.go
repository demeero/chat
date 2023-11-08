package cassandra

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/demeero/chat/bricks/logger"
	"github.com/demeero/chat/bricks/meter"
	"github.com/gocql/gocql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
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
	span.SetAttributes(semconv.DBStatementKey.String(q.Statement),
		semconv.DBSystemCassandra,
		attribute.String("keyspace", q.Keyspace),
		attribute.Int("rows", q.Rows),
		attribute.Int("attempt", q.Attempt))
	if q.Err != nil && !errors.Is(q.Err, gocql.ErrNotFound) {
		span.RecordError(q.Err)
		span.SetStatus(codes.Error, "")
	}
	span.SetStatus(codes.Ok, "")
	span.End(trace.WithTimestamp(q.End.UTC()))
}

type queryMetrics struct {
	latencyHist  metric.Int64Histogram
	queryCounter metric.Int64Counter
}

func newQueryMetrics() (*queryMetrics, error) {
	cqlMeter := otel.GetMeterProvider().Meter("cql")
	latency, err := cqlMeter.Int64Histogram("cql_query_latency", metric.WithDescription("cql query latency"), metric.WithUnit("ms"))
	if err != nil {
		return nil, fmt.Errorf("failed create cql_query_latency metric: %w", err)
	}
	queryCounter, err := cqlMeter.Int64Counter("cql_query_count", metric.WithDescription("cql query count"))
	if err != nil {
		return nil, fmt.Errorf("failed create cql_query_count metric: %w", err)
	}
	return &queryMetrics{
		latencyHist:  latency,
		queryCounter: queryCounter,
	}, nil
}

type MeterQueryObserver struct {
	Disabled bool
	qMeter   *queryMetrics
}

func (o MeterQueryObserver) ObserveQuery(ctx context.Context, q gocql.ObservedQuery) {
	if o.Disabled {
		return
	}
	if o.qMeter == nil {
		qMeter, err := newQueryMetrics()
		if err != nil {
			logger.FromCtx(ctx).Error("failed create cql query metrics", slog.Any("err", err))
			return
		}
		o.qMeter = qMeter
	}
	attrs := append(meter.AttrsFromCtx(ctx), semconv.DBSystemCassandra)
	if q.Err != nil && !errors.Is(q.Err, gocql.ErrNotFound) {
		attrs = append(attrs, semconv.OTelStatusCodeError)
	} else {
		attrs = append(attrs, semconv.OTelStatusCodeOk)
	}
	if q.Attempt > 0 {
		attrs = append(attrs, attribute.Bool("with_retry", true))
	}
	attrs = append(attrs, attribute.String("keyspace", q.Keyspace))
	o.qMeter.queryCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	o.qMeter.latencyHist.Record(ctx, q.Metrics.TotalLatency/1e6, metric.WithAttributes(attrs...))
}
