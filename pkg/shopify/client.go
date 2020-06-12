package shopify

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/peterhellberg/link"

	"github.com/demosdemon/shop/pkg/log"
	"github.com/demosdemon/shop/pkg/retry"
)

const (
	DefaultAPIVersion  = "2020-04"
	DefaultUserAgent   = "shop/1.0.0"
	DefaultHTTPTimeout = 5 * time.Minute
	DefaultRetryCount  = 10
	DefaultRetryDelay  = 100 * time.Millisecond
	DefaultRetryJitter = 100 * time.Millisecond

	fmtBaseURL = "https://%s.myshopify.com"
	pathPrefix = "admin/api"

	mApplicationJSON = "application/json"

	hAPICallLimit = "X-Shopify-Shop-Api-Call-Limit"
	hAccept       = "Accept"
	hContentType  = "Content-Type"
	hRetryAfter   = "Retry-After"
	hUserAgent    = "User-Agent"
)

func New(storeID, username, password string, options ...Option) *Client {
	c := &Client{
		Client:   http.Client{Timeout: DefaultHTTPTimeout},
		storeID:  storeID,
		username: username,
		password: password,
	}

	for _, opt := range options {
		opt(c)
	}

	return c
}

type Client struct {
	http.Client

	storeID  string
	username string
	password string

	apiVersion  *string
	userAgent   *string
	retryCount  *int
	retryDelay  *time.Duration
	retryJitter *time.Duration
	logger      log.Logger

	rateLimitInfo RateLimitInfo
}

func (c *Client) BaseURL() (*url.URL, error) {
	s := fmt.Sprintf(fmtBaseURL, c.storeID)
	return url.Parse(s)
}

func (c *Client) StoreID() string {
	return c.storeID
}

func (c *Client) APIVersion() string {
	s := c.apiVersion
	if s == nil {
		return DefaultAPIVersion
	}
	return *s
}

func (c *Client) UserAgent() string {
	s := c.userAgent
	if s == nil {
		return DefaultUserAgent
	}
	return *s
}

func (c *Client) RetryCount() int {
	i := c.retryCount
	if i == nil {
		return DefaultRetryCount
	}
	return *i
}

func (c *Client) RetryDelay() time.Duration {
	d := c.retryDelay
	if d == nil {
		return DefaultRetryDelay
	}
	return *d
}

func (c *Client) RetryJitter() time.Duration {
	d := c.retryJitter
	if d == nil {
		return DefaultRetryJitter
	}
	return *d
}

func (c *Client) Deserialize(res *http.Response, resource interface{}) error {
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if err := res.Body.Close(); err != nil {
		return err
	}
	if err := json.Unmarshal(data, resource); err != nil {
		return NewResponseDecodingError(res, err, data)
	}
	return nil
}

func (c *Client) Paginate(ctx context.Context, element string, options interface{}) (<-chan PaginationResult, error) {
	limit, ok := getLimit(options)
	if !ok {
		limit = 50
	}

	count, err := c.Count(ctx, c.Path(element), options)
	if err != nil {
		return nil, err
	}
	c.Infof("expecting %d records", count)

	pages := int(math.Ceil(float64(count) / float64(limit)))

	ch := make(chan PaginationResult)
	records := 0
	go func() {
		defer close(ch)
		relPath := c.Path(element) + ".json"

		page := 0
		for {
			page++
			c.Infof("fetching %s page %d of %d", element, page, pages)
			res, err := c.Get(ctx, relPath, options)
			if err == context.Canceled || err == context.DeadlineExceeded {
				return
			}

			if err != nil {
				ch <- PaginationResult{err: err}
				return
			}

			var resource map[string][]json.RawMessage
			if err := c.Deserialize(res, &resource); err != nil {
				ch <- PaginationResult{err: err}
				return
			}

			values := resource[element]
			for _, value := range values {
				records++
				ch <- PaginationResult{msg: value}
			}

			options, err = getNextPageOptions(res)
			if err != nil {
				data := []byte(url.Values(res.Header).Encode())
				err := NewResponseDecodingError(res, err, data)
				ch <- PaginationResult{err: err}
				return
			}

			if options == nil {
				if count != records {
					c.Warnf("expected %d records but got %d", count, records)
				}
				return
			}

			if v, ok := options.(url.Values); ok {
				if v == nil {
					return
				}

				if pi := v.Get("page_info"); pi != "" {
					dec, err := base64.RawStdEncoding.DecodeString(pi)
					if err != nil {
						c.Warnf("error decoding page info: %v", err)
						continue
					}
					c.Debugf("next page info: %s", string(dec))
				}
			}
		}
	}()

	return ch, nil
}

func (c *Client) Count(ctx context.Context, path string, options interface{}) (int, error) {
	var resource struct {
		Count int `json:"count"`
	}

	res, err := c.Get(ctx, path+"/count.json", options)
	if err != nil {
		return 0, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}

	if err := res.Body.Close(); err != nil {
		return 0, err
	}

	if err := json.Unmarshal(body, &resource); err != nil {
		return 0, NewResponseDecodingError(res, err, body)
	}

	return resource.Count, nil
}

func (c *Client) Path(relPath string) string {
	relPath = strings.TrimLeft(relPath, "/")
	relPath = path.Join(pathPrefix, c.APIVersion(), relPath)
	return relPath
}

func (c *Client) Get(ctx context.Context, path string, options interface{}) (*http.Response, error) {
	return c.CreateAndDo(ctx, http.MethodGet, path, nil, options)
}

func (c *Client) Post(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	return c.CreateAndDo(ctx, http.MethodPost, path, body, nil)
}

func (c *Client) Put(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	return c.CreateAndDo(ctx, http.MethodPut, path, body, nil)
}

func (c *Client) Delete(ctx context.Context, path string) (*http.Response, error) {
	return c.CreateAndDo(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) CreateAndDo(ctx context.Context, method, path string, body, options interface{}) (*http.Response, error) {
	res, err := c.NewRequest(ctx, method, path, body, options)
	if err != nil {
		return nil, err
	}
	return c.Do(res)
}

func (c *Client) NewRequest(ctx context.Context, method, path string, body, options interface{}) (*http.Request, error) {
	baseUrl, err := c.BaseURL()
	if err != nil {
		return nil, err
	}

	rel, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	u, err := appendQuery(baseUrl.ResolveReference(rel), options)
	if err != nil {
		return nil, err
	}

	b, err := marshalBody(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), b)
	if err != nil {
		return nil, err
	}

	req.Header.Add(hContentType, mApplicationJSON)
	req.Header.Add(hAccept, mApplicationJSON)
	req.Header.Add(hUserAgent, c.UserAgent())
	if u.Host == baseUrl.Host {
		req.SetBasicAuth(c.username, c.password)
	}

	return req, nil
}

func (c *Client) Do(req *http.Request) (res *http.Response, err error) {
	retryCount := c.RetryCount()
	delay := delay(c.RetryDelay(), c.RetryJitter())
	c.logRequest(req)

	err = retry.Go(
		func() (err error) {
			res, err = c.Client.Do(req)
			c.logResponse(res)
			if err == nil {
				err = CheckResponseError(res)
			}
			if err := c.rateLimitInfo.update(res); err != nil {
				c.Warnf("error updating rate limit info: %v", err)
			}
			return
		},
		retry.WithMaxAttempts(retryCount),
		retry.WithOnRetry(func(attempt int, wait time.Duration, err error) {
			c.Infof("attempt %d/%d: %v; sleeping %s", attempt, retryCount, err, wait)
		}),
		retry.WithDoRetryWithDelay(func(attempt int, err error) (time.Duration, bool) {
			if err == context.Canceled || err == context.DeadlineExceeded {
				return 0, false
			}
			if rateLimitErr, ok := err.(RateLimitError); ok {
				return rateLimitErr.RetryAfter, ok
			}
			if resErr, ok := err.(ResponseError); ok {
				return delay(attempt),
					!(http.StatusBadRequest <= resErr.Status && resErr.Status < http.StatusInternalServerError)
			}
			return delay(attempt), true
		}),
	)

	return
}

func (c *Client) logRequest(req *http.Request) {
	if req == nil {
		return
	}
	if req.URL != nil {
		c.Infof("%s: %s", req.Method, req.URL)
	}
	c.logBody(&req.Body, "SENT: %s")
}

func (c *Client) logResponse(res *http.Response) {
	if res == nil {
		c.Debugf("nil response")
		return
	}
	c.Debugf("RECV %03d: %s", res.StatusCode, res.Status)
	c.logBody(&res.Body, "RESP: %s")
}

func (c *Client) logBody(body *io.ReadCloser, format string) {
	if body == nil {
		return
	}
	if *body == nil {
		return
	}
	data, _ := ioutil.ReadAll(*body)
	if len(data) > 0 {
		c.Tracef(format, string(data))
	}
	*body = ioutil.NopCloser(bytes.NewReader(data))
}

func (c *Client) Logf(level log.Level, format string, v ...interface{}) {
	l := c.logger
	if l == nil {
		return
	}

	l.Logf(level, format, v...)
}

func (c *Client) Errorf(format string, v ...interface{}) {
	c.Logf(log.LevelError, format, v...)
}

func (c *Client) Warnf(format string, v ...interface{}) {
	c.Logf(log.LevelWarn, format, v...)
}

func (c *Client) Infof(format string, v ...interface{}) {
	c.Logf(log.LevelInfo, format, v...)
}

func (c *Client) Debugf(format string, v ...interface{}) {
	c.Logf(log.LevelDebug, format, v...)
}

func (c *Client) Tracef(format string, v ...interface{}) {
	c.Logf(log.LevelTrace, format, v...)
}

func delay(initial, jitter time.Duration) func(step int) time.Duration {
	return func(step int) time.Duration {
		n := initial
		n *= 1 << step
		n += time.Duration(rand.Int63n(int64(jitter)))
		return n
	}
}

func appendQuery(u *url.URL, v interface{}) (*url.URL, error) {
	if v == nil {
		return u, nil
	}

	q, err := query.Values(v)
	if err != nil {
		var ok bool
		q, ok = v.(url.Values)
		if !ok {
			return nil, err
		}
	}

	for k, values := range u.Query() {
		for _, v := range values {
			q.Add(k, v)
		}
	}

	c := cloneURL(u)
	c.RawQuery = q.Encode()
	return c, nil
}

func marshalBody(v interface{}) (io.Reader, error) {
	if v == nil {
		return nil, nil
	}

	buf, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(buf), nil
}

func cloneURL(u *url.URL) *url.URL {
	c := *u
	if u.User != nil {
		u := *u.User
		c.User = &u
	}
	return &c
}

func wrapSpecificError(r *http.Response, err ResponseError) error {
	// see https://www.shopify.dev/concepts/about-apis/response-codes
	if err.Status == http.StatusTooManyRequests {
		f, fe := retryAfter(r)
		if fe != nil {
			return fe
		}
		return RateLimitError{
			ResponseError: err,
			RetryAfter:    f,
		}
	}

	// if err.Status == http.StatusSeeOther {
	// todo
	// The response to the request can be found under a different URL in the
	// Location header and can be retrieved using a GET method on that resource.
	// }

	if err.Status == http.StatusNotAcceptable {
		err.Message = http.StatusText(err.Status)
	}

	return err
}

func retryAfter(res *http.Response) (time.Duration, error) {
	const bits = 64
	h := res.Header.Get(hRetryAfter)
	if h == "" {
		return 0, nil
	}
	f, err := strconv.ParseFloat(h, bits)
	if err != nil {
		return 0, err
	}
	return time.Duration(float64(time.Second) * f), nil
}

func getNextPageOptions(res *http.Response) (url.Values, error) {
	g := link.ParseResponse(res)
	l := g["next"]

	if l == nil {
		return nil, nil
	}

	rel, err := url.Parse(l.URI)
	if err != nil {
		return nil, err
	}

	q, err := url.ParseQuery(rel.RawQuery)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func getLimit(options interface{}) (limit int, ok bool) {
	if options == nil {
		return
	}

	q, err := query.Values(options)
	if err != nil {
		q, ok = options.(url.Values)
	}
	limit, err = strconv.Atoi(q.Get("limit"))
	ok = err == nil
	return
}
