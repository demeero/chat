package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"github.com/demeero/chat/bricks/httpsrv"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

func setupHTTPSrv(ctx context.Context, cfg Config, l Loader) *echo.Echo {
	meterMW, err := httpsrv.Meter()
	if err != nil {
		log.Fatalf("failed create meter middleware: %s", err)
	}
	httpCfg := cfg.HTTP
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

	e.GET("/history", func(c echo.Context) error {
		msgs, err := l.Load(c.Request().Context())
		if err != nil {
			return fmt.Errorf("failed load chat history: %w", err)
		}
		if msgs == nil {
			msgs = []Message{}
		}
		return c.JSON(http.StatusOK, msgs)
	})

	go func() {
		slog.Info("initializing HTTP server", slog.Int("port", httpCfg.Port))
		if err := e.Start(fmt.Sprintf(":%d", httpCfg.Port)); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed http server: %s", err)
		}
	}()

	return e
}
