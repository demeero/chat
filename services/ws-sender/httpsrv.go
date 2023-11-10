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
	"github.com/demeero/chat/bricks/session"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"golang.org/x/net/websocket"
)

const topic = "msg_sent"

func setupHTTPSrv(ctx context.Context, cfg Config, pub message.Publisher) *echo.Echo {
	meterMW, err := httpsrv.Meter()
	if err != nil {
		log.Fatalf("failed create meter middleware: %s", err)
	}
	e := httpsrv.Configure(httpsrv.Config{
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		Port:              cfg.HTTP.Port,
	})
	e.Pre(echomw.RemoveTrailingSlash())
	e.Use(httpsrv.RecoverMW())
	e.Use(otelecho.Middleware(cfg.ServiceName))
	e.Use(meterMW)
	e.Use(httpsrv.LogCtxMW())
	e.Use(httpsrv.TokenMW(ctx, cfg.JwksURL))
	e.Use(httpsrv.LogMW(slog.LevelDebug))

	e.GET("/sender", sender(ctx, pub))

	go func() {
		slog.Info("initializing HTTP server", slog.Int("port", cfg.HTTP.Port))
		if err := e.Start(fmt.Sprintf(":%d", cfg.HTTP.Port)); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed http server: %s", err)
		}
	}()

	return e
}

func sender(ctx context.Context, pub message.Publisher) echo.HandlerFunc {
	return func(c echo.Context) error {
		websocket.Handler(func(ws *websocket.Conn) {
			go func() {
				<-ctx.Done()
				ws.Close()
			}()
			Sender{
				Topic: topic,
				Sess:  session.FromCtx(c.Request().Context()),
				Pub:   pub,
			}.Execute(ws)
		}).ServeHTTP(c.Response(), c.Request())
		return nil
	}
}
