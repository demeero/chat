package httpsrv

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/demeero/chat/bricks/apperr"
	"github.com/demeero/chat/bricks/logger"
	"github.com/labstack/echo/v4"
	echolog "github.com/labstack/gommon/log"
)

type Config struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	Port              int
}

func Configure(cfg Config) *echo.Echo {
	e := echo.New()
	srv := e.Server
	srv.WriteTimeout = cfg.WriteTimeout
	srv.ReadTimeout = cfg.ReadTimeout
	srv.ReadHeaderTimeout = cfg.ReadHeaderTimeout
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = errorHandler
	e.Logger.SetLevel(echolog.OFF)
	return e
}

func errorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	var (
		lg      = logger.FromCtx(c.Request().Context())
		echoErr *echo.HTTPError
	)
	switch {
	case errors.As(err, &echoErr):
		handleEchoErr(echoErr, lg)
	case errors.Is(err, apperr.ErrInvalidData):
		echoErr = echo.NewHTTPError(http.StatusBadRequest, err.Error())
	case errors.Is(err, apperr.ErrNotFound):
		echoErr = echo.NewHTTPError(http.StatusNotFound, err.Error())
	case errors.Is(err, apperr.ErrForbidden):
		echoErr = echo.NewHTTPError(http.StatusForbidden, err.Error())
	case errors.Is(err, apperr.ErrConflict):
		echoErr = echo.NewHTTPError(http.StatusConflict, err.Error())
	case errors.Is(err, apperr.ErrUnauthorized):
		echoErr = echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	default:
		slog.Error("internal server error", slog.Any("err", err))
		echoErr = echo.NewHTTPError(http.StatusInternalServerError)
	}
	if err = c.JSON(echoErr.Code, echoErr); err != nil {
		slog.Error("failed send error response", slog.Any("err", err))
	}
}

func handleEchoErr(echoErr *echo.HTTPError, lg *slog.Logger) {
	if echoErr.Internal != nil {
		lg.Error("failed to handle request", slog.Any("err", echoErr.Internal))
	}
	if msg, ok := echoErr.Message.(string); ok && msg == "" {
		echoErr.Message = http.StatusText(echoErr.Code)
	}
}
