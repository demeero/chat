package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/demeero/chat/bricks/httpsrv"
	"github.com/demeero/chat/bricks/logger"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"golang.org/x/net/websocket"
)

const topic = "msg_sent"

func setupHTTPSrv(ctx context.Context, cfg Config, sub message.Subscriber) *echo.Echo {
	httpCfg := cfg.HTTP
	meterMW, err := httpsrv.Meter()
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
	corsCfg := echomw.DefaultCORSConfig
	corsCfg.AllowOrigins = httpCfg.CORS.AllowedOrigins
	corsCfg.AllowCredentials = true
	e.Use(echomw.CORSWithConfig(corsCfg))
	e.Use(httpsrv.RecoverMW())
	e.Use(otelecho.Middleware(cfg.ServiceName))
	e.Use(meterMW)
	e.Use(httpsrv.LogCtxMW())
	e.Use(httpsrv.TokenMW(ctx, cfg.JwksURL))
	e.Use(httpsrv.LogMW(slog.LevelDebug))

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
				logger.FromCtx(c.Request().Context()).Error("failed subscribe", slog.Any("err", err))
			}
		}).ServeHTTP(c.Response(), c.Request())
		return nil
	}
}
