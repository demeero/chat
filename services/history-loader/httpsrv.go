package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"slices"
	"strconv"

	"github.com/demeero/chat/bricks/apperr"
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
	e.Use(httpsrv.RecoverMW())
	e.Use(otelecho.Middleware(cfg.ServiceName))
	e.Use(meterMW)
	e.Use(httpsrv.LogCtxMW())
	e.Use(httpsrv.TokenMW(ctx, cfg.JwksURL))
	e.Use(httpsrv.LogMW(slog.LevelDebug))

	e.GET("/history", func(c echo.Context) error {
		pSize := c.QueryParam("page_size")
		if pSize == "" {
			pSize = "0"
		}
		pSizeInt, err := strconv.Atoi(pSize)
		if err != nil {
			return fmt.Errorf("%w: failed parse page size: %s", apperr.ErrInvalidData, err)
		}
		p, err := NewPagination(c.QueryParam("page_token"), uint16(pSizeInt))
		if err != nil {
			return err
		}
		msgs, pt, err := l.Load(c.Request().Context(), p)
		if err != nil {
			return fmt.Errorf("failed load chat history: %w", err)
		}
		if msgs == nil {
			msgs = []Message{}
		}
		slices.Reverse(msgs)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"page":            msgs,
			"next_page_token": pt,
		})
	})

	go func() {
		slog.Info("initializing HTTP server", slog.Int("port", httpCfg.Port))
		if err := e.Start(fmt.Sprintf(":%d", httpCfg.Port)); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed http server: %s", err)
		}
	}()

	return e
}
