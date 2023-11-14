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
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/demeero/bricks/echobrick"
	"github.com/demeero/bricks/errbrick"
	"github.com/demeero/chat/bricks/httpsrv"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

func setupHTTPSrv(ctx context.Context, cfg Config, l Loader) *echo.Echo {
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

	e.GET("/history", func(c echo.Context) error {
		pSize := c.QueryParam("page_size")
		if pSize == "" {
			pSize = "0"
		}
		pSizeInt, err := strconv.Atoi(pSize)
		if err != nil {
			return fmt.Errorf("%w: failed parse page size: %s", errbrick.ErrInvalidData, err)
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
