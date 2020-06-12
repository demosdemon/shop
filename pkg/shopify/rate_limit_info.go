package shopify

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type RateLimitInfo struct {
	RequestCount int
	BucketSize   int
	RetryAfter   time.Duration
}

func (rl *RateLimitInfo) update(res *http.Response) error {
	if res == nil {
		return errors.New("nil response")
	}
	const sep = "/"
	var err error

	if s := strings.Split(res.Header.Get(hAPICallLimit), sep); len(s) == 2 {
		rl.RequestCount, err = strconv.Atoi(s[0])
		if err != nil {
			return errors.Wrap(err, "error converting request count to an integer")
		}

		rl.BucketSize, err = strconv.Atoi(s[1])
		if err != nil {
			return errors.Wrap(err, "error converting bucket size to an integer")
		}
	}

	rl.RetryAfter, err = retryAfter(res)
	if err != nil {
		return errors.Wrap(err, "error converting retry after to a duration")
	}

	return nil
}
