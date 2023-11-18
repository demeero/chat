package loader

import (
	"encoding/base64"
	"fmt"

	"github.com/demeero/bricks/errbrick"
)

// Pagination is the pagination parameters.
type Pagination struct {
	// PageToken is the token to get the next page.
	pageToken []byte
	// PageSize is the number of items per page.
	pageSize uint16
}

func NewPagination(pageToken string, pageSize uint16) (Pagination, error) {
	if pageSize < 1 {
		pageSize = 30
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	var tokenBytes []byte
	if pageToken != "" {
		b, err := base64.StdEncoding.DecodeString(pageToken)
		if err != nil {
			return Pagination{}, fmt.Errorf("%w: failed to decode token from base64: %s", errbrick.ErrInvalidData, err)
		}
		tokenBytes = b
	}
	return Pagination{
		pageSize:  pageSize,
		pageToken: tokenBytes,
	}, nil
}
