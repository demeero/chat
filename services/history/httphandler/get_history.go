package httphandler

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/demeero/bricks/errbrick"
	"github.com/demeero/chat/history/loader"
	"github.com/labstack/echo/v4"
)

func GetHistory(l *loader.Loader) func(c echo.Context) error {
	return func(c echo.Context) error {
		pSize := c.QueryParam("page_size")
		if pSize == "" {
			pSize = "0"
		}
		pSizeInt, err := strconv.Atoi(pSize)
		if err != nil {
			return fmt.Errorf("%w: failed parse page size: %s", errbrick.ErrInvalidData, err)
		}
		p, err := loader.NewPagination(c.QueryParam("page_token"), uint16(pSizeInt))
		if err != nil {
			return err
		}
		msgs, pt, err := l.Load(c.Request().Context(), c.Param("room_chat_id"), p)
		if err != nil {
			return fmt.Errorf("failed load chat history: %w", err)
		}
		if msgs == nil {
			msgs = []loader.Message{}
		}
		slices.Reverse(msgs)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"page":            msgs,
			"next_page_token": pt,
		})
	}
}
