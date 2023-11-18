package httphandler

import (
	"context"
	"log"
	"log/slog"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/demeero/bricks/echobrick"
	"github.com/demeero/chat/bricks/httpsrv"
	"github.com/demeero/chat/history/loader"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

func Setup(ctx context.Context, jwksURL, serviceName string, e *echo.Echo, l *loader.Loader) error {
	meterMW, err := echobrick.OTELMeterMW(echobrick.OTELMeterMWConfig{
		Attrs: &echobrick.OTELMeterAttrsConfig{
			Method:     true,
			Path:       true,
			Status:     true,
			AttrsToCtx: true,
		},
		Metrics: &echobrick.OTELMeterMetricsConfig{
			ReqDuration:       true,
			ReqCounter:        true,
			ActiveReqsCounter: true,
			ReqSize:           true,
			RespSize:          true,
		},
	})
	if err != nil {
		log.Fatalf("failed create meter middleware: %s", err)
	}
	e.Pre(echomw.RemoveTrailingSlash())
	e.Use(echobrick.RecoverSlogMW())
	e.Use(otelecho.Middleware(serviceName))
	e.Use(meterMW)
	e.Use(echobrick.SlogCtxMW(echobrick.LogCtxMWConfig{Trace: true}))
	e.Use(echobrick.TokenClaimsMW(jwksURL, keyfunc.Options{
		Ctx: ctx,
		RefreshErrorHandler: func(err error) {
			slog.Error("failed to refresh jwks", slog.Any("err", err))
		},
		RefreshInterval:   time.Minute,
		RefreshRateLimit:  time.Second * 20,
		RefreshTimeout:    time.Second * 10,
		RefreshUnknownKID: true,
	}))
	e.Use(httpsrv.SessionCtxMW())
	e.Use(echobrick.SlogLogMW(slog.LevelDebug, nil))
	e.GET("/:room_chat_id", GetHistory(l))

	for _, r := range e.Routes() {
		if r != nil {
			slog.Info("registered route",
				slog.String("method", r.Method), slog.String("path", r.Path), slog.String("name", r.Name))
		}
	}

	return nil
}
