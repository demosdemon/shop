package shopify

import (
	"time"

	"github.com/demosdemon/shop/pkg/log"
)

type Option func(c *Client)

func WithAPIVersion(apiVersion string) Option {
	return func(c *Client) {
		c.apiVersion = &apiVersion
	}
}

func WithUserAgent(userAgent string) Option {
	return func(c *Client) {
		c.userAgent = &userAgent
	}
}

func WithHTTPTimeout(httpTimeout time.Duration) Option {
	return func(c *Client) {
		c.Timeout = httpTimeout
	}
}

func WithRetryCount(retryCount int) Option {
	return func(c *Client) {
		c.retryCount = &retryCount
	}
}

func WithRetryDelay(retryDelay time.Duration) Option {
	return func(c *Client) {
		c.retryDelay = &retryDelay
	}
}

func WithRetryJitter(retryJitter time.Duration) Option {
	return func(c *Client) {
		c.retryJitter = &retryJitter
	}
}

func WithLogger(logger log.Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}

// General list options that can be used for most collections of entities.
type ListOptions struct {
	// PageInfo is used with new pagination search.
	PageInfo string `url:"page_info,omitempty"`

	// Page is used to specify a specific page to load.
	// It is the deprecated way to do pagination.
	Page         int       `url:"page,omitempty"`
	Limit        int       `url:"limit,omitempty"`
	SinceID      int64     `url:"since_id,omitempty"`
	CreatedAtMin time.Time `url:"created_at_min,omitempty"`
	CreatedAtMax time.Time `url:"created_at_max,omitempty"`
	UpdatedAtMin time.Time `url:"updated_at_min,omitempty"`
	UpdatedAtMax time.Time `url:"updated_at_max,omitempty"`
	Order        string    `url:"order,omitempty"`
	Fields       string    `url:"fields,omitempty"`
	Vendor       string    `url:"vendor,omitempty"`
	IDs          []int64   `url:"ids,omitempty,comma"`
}

// General count options that can be used for most collection counts.
type CountOptions struct {
	CreatedAtMin time.Time `url:"created_at_min,omitempty"`
	CreatedAtMax time.Time `url:"created_at_max,omitempty"`
	UpdatedAtMin time.Time `url:"updated_at_min,omitempty"`
	UpdatedAtMax time.Time `url:"updated_at_max,omitempty"`
}
