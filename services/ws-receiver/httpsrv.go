package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/demeero/bricks/echobrick"
	"github.com/demeero/bricks/slogbrick"
	"github.com/demeero/chat/bricks/httpsrv"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"golang.org/x/net/websocket"
)

const topic = "msg_sent"

func setupHTTPSrv(ctx context.Context, cfg Config, sub message.Subscriber) *echo.Echo {
	httpCfg := cfg.HTTP
	meterMW, err := echobrick.OTELMeterMW(echobrick.OTELMeterMWConfig{
		Attrs: &echobrick.OTELMeterAttrsConfig{
			Method:     true,
			Path:       true,
			Status:     true,
			AttrsToCtx: true,
		},
		Metrics: &echobrick.OTELMeterMetricsConfig{
			LatencyHist:  true,
			ReqCounter:   true,
			ReqSizeHist:  true,
			RespSizeHist: true,
		},
	})
	if err != nil {
		log.Fatalf("failed create meter middleware: %s", err)
	}
	e := httpsrv.Configure(httpsrv.Config{
		ReadHeaderTimeout: httpCfg.ReadHeaderTimeout,
		ReadTimeout:       httpCfg.ReadTimeout,
		WriteTimeout:      httpCfg.WriteTimeout,
		Port:              httpCfg.Port,
	})
	e.Pre(echomw.RemoveTrailingSlash())
	e.Use(echobrick.RecoverSlogMW())
	e.Use(otelecho.Middleware(cfg.ServiceName))
	e.Use(meterMW)
	e.Use(echobrick.SlogCtxMW(echobrick.LogCtxMWConfig{Trace: true}))
	e.Use(echobrick.TokenClaimsMW(cfg.JwksURL, keyfunc.Options{
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

	e.GET("/receiver", receiverHandler(sub))

	go func() {
		slog.Info("initializing HTTP server", slog.Int("port", httpCfg.Port))
		if err := e.Start(fmt.Sprintf(":%d", httpCfg.Port)); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed http server: %s", err)
		}
	}()

	return e
}

func receiverHandler(sub message.Subscriber) echo.HandlerFunc {
	return func(c echo.Context) error {
		websocket.Handler(func(ws *websocket.Conn) {
			defer ws.Close()
			err := Subscriber{
				Topic: topic,
				Sub:   sub,
			}.Subscribe(ws.Request().Context(), ws)
			if err != nil {
				slogbrick.FromCtx(c.Request().Context()).Error("failed subscribe", slog.Any("err", err))
			}
		}).ServeHTTP(c.Response(), c.Request())
		return nil
	}
}
