package httpsrv

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/demeero/chat/bricks/logger"
	"github.com/demeero/chat/bricks/meter"
	"github.com/demeero/chat/bricks/session"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// LogCtxMW is a middleware that adds logger to request context.
func LogCtxMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			reqLogger := slog.Default().With(
				slog.String("http_route", req.RequestURI),
				slog.String("http_method", req.Method))
			ctx := logger.ToCtx(req.Context(), reqLogger)
			c.SetRequest(req.WithContext(ctx))
			return next(c)
		}
	}
}

// LogMW is a middleware to provide logging for each request.
// It logs the request URI, method, host, remote address, real IP, user agent, request ID, duration and response size.
func LogMW(lvl slog.Level) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			reqLogger := logger.FromCtx(req.Context())

			reqLogger.Log(req.Context(), lvl, "received request")

			start := time.Now().UTC()

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			res := c.Response()

			reqLogger.Log(req.Context(), lvl, "completed handling request",
				slog.Duration("req_duration", time.Since(start)), slog.Int("resp_status", res.Status))
			return err
		}
	}
}

// RecoverMW recovers from panics and logs the stack trace.
// It returns a 500 status code.
func RecoverMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if err := recover(); err != nil {
					c.Response().WriteHeader(http.StatusInternalServerError)
					logger.FromCtx(c.Request().Context()).
						Error("http handler panicked", slog.Any("err", err), slog.String("stack", string(debug.Stack())))
				}
			}()
			return next(c)
		}
	}
}

// TokenClaims is a custom jwt token claims.
type TokenClaims struct {
	jwt.RegisteredClaims
	Session session.Session `json:"session"`
}

func TokenMW(ctx context.Context, jwksURL string) echo.MiddlewareFunc {
	var jwtKeyFunc jwt.Keyfunc = func(token *jwt.Token) (interface{}, error) {
		return jwt.UnsafeAllowNoneSignatureType, nil
	}
	if jwksURL != "" {
		options := keyfunc.Options{
			Ctx: ctx,
			RefreshErrorHandler: func(err error) {
				slog.Error("there was an error with the jwt.Keyfunc", slog.Any("err", err))
			},
			RefreshInterval:   time.Minute,
			RefreshRateLimit:  time.Second * 20,
			RefreshTimeout:    time.Second * 10,
			RefreshUnknownKID: true,
		}
		jwks, err := keyfunc.Get(jwksURL, options)
		if err != nil {
			slog.Error("failed to create JWKS from resource at the given URL",
				slog.Any("err", err), slog.String("jwksURL", jwksURL))
		} else {
			jwtKeyFunc = jwks.Keyfunc
		}
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			jwtToken, err := retrieveJWT(c.Request())
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, err)
			}
			claims := TokenClaims{}
			tkn, err := jwt.ParseWithClaims(jwtToken, &claims, jwtKeyFunc)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "failed to parse token")
			}
			if !tkn.Valid {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}
			ctx := session.ToCtx(c.Request().Context(), claims.Session)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

// retrieveJWT returns the token string from the request.
func retrieveJWT(request *http.Request) (string, error) {
	header := request.Header.Get("Authorization")
	if header == "" {
		return "", errors.New("authorization header is empty")
	}
	h := strings.Split(header, " ")
	if len(h) != 2 || !strings.EqualFold(h[0], "bearer") {
		return "", errors.New("invalid authorization header format")
	}
	return h[1], nil
}

// Meter is a middleware that records metrics for each request.
func Meter() (echo.MiddlewareFunc, error) {
	httpMeter := otel.GetMeterProvider().Meter("http-server")
	srvLatencyHist, err := httpMeter.Int64Histogram("http_server_latency",
		metric.WithDescription("The latency of HTTP requests"), metric.WithUnit("ms"))
	if err != nil {
		return nil, fmt.Errorf("failed create http_server_latency metric: %w", err)
	}
	srvReqCounter, err := httpMeter.Int64Counter("http_server_request_count",
		metric.WithDescription("The number of HTTP requests"))
	if err != nil {
		return nil, fmt.Errorf("failed create http_server_request_count metric: %w", err)
	}
	srvReqSizeHist, err := httpMeter.Int64Histogram("http_server_request_size",
		metric.WithDescription("The size of HTTP requests"), metric.WithUnit("B"))
	if err != nil {
		return nil, fmt.Errorf("failed create http_server_request_size metric: %w", err)
	}
	srvRespSizeHist, err := httpMeter.Int64Histogram("http_server_response_size",
		metric.WithDescription("The size of HTTP responses"), metric.WithUnit("B"))
	if err != nil {
		return nil, fmt.Errorf("failed create http_server_response_size metric: %w", err)
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now().UTC()
			reqSz := computeApproximateRequestSize(c.Request())

			attrs := []attribute.KeyValue{
				attribute.String("method", c.Request().Method),
				attribute.String("path", c.Path()),
			}

			c.SetRequest(c.Request().WithContext(meter.AttrsToCtx(c.Request().Context(), attrs)))

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			attrs = append(attrs, attribute.Int("status", c.Response().Status))
			ctx := c.Request().Context()

			srvReqCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
			srvLatencyHist.Record(ctx, time.Since(start).Milliseconds(), metric.WithAttributes(attrs...))
			srvReqSizeHist.Record(ctx, reqSz, metric.WithAttributes(attrs...))
			srvRespSizeHist.Record(ctx, c.Response().Size, metric.WithAttributes(attrs...))
			return err
		}
	}, nil
}

func computeApproximateRequestSize(r *http.Request) int64 {
	s := 0
	if r.URL != nil {
		s = len(r.URL.Path)
	}

	s += len(r.Method)
	s += len(r.Proto)
	for name, values := range r.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}
	s += len(r.Host)

	// N.B. r.Form and r.MultipartForm are assumed to be included in r.URL.

	if r.ContentLength != -1 {
		s += int(r.ContentLength)
	}
	return int64(s)
}
