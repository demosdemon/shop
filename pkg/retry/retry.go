package retry

import (
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const (
	DefaultMaxAttempts = 10
	DefaultDelay       = time.Millisecond * 100
)

var (
	DefaultDoRetry = func(attempt int, err error) (time.Duration, bool) { return DefaultDelay, true }
)

type Retryable interface {
	Do() error
}

type RetryableAttempts interface {
	Retryable
	MaxAttempts() int
}

type RetryableDoRetry interface {
	Retryable
	DoRetry(err error) bool
}

type RetryableDoRetryWithDelay interface {
	Retryable
	DoRetry(attempt int, err error) (time.Duration, bool)
}

type RetryableOnRetry interface {
	Retryable
	OnRetry(attempt int, wait time.Duration, err error)
}

func Go(do func() error, options ...Option) error {
	r := New(do, options...)
	return Do(r)
}

func Do(retryable Retryable) error {
	if retryable == nil {
		return errors.New("retryable is nil")
	}

	var (
		do          = retryable.Do
		doRetry     = doRetry(retryable)
		onRetry     = onRetry(retryable)
		maxAttempts = maxAttempts(retryable)
	)

	if maxAttempts < 1 {
		return errors.New("maxAttempts is less than 1")
	}

	var multi multierror.Error
	attempts := 0
	for {
		attempts++
		err := do()
		if err == nil {
			return nil
		}

		multi.Errors = append(multi.Errors, err)
		wait, next := doRetry(attempts, err)
		if !(next && attempts < maxAttempts) {
			return multi.Unwrap()
		}
		onRetry(attempts, wait, err)
		time.Sleep(wait)
	}
}

func maxAttempts(retryable Retryable) int {
	if r, ok := retryable.(RetryableAttempts); ok {
		return r.MaxAttempts()
	}

	return DefaultMaxAttempts
}

func doRetry(retryable Retryable) func(int, error) (time.Duration, bool) {
	switch r := retryable.(type) {
	case RetryableDoRetry:
		return adaptDoRetry(r.DoRetry)
	case RetryableDoRetryWithDelay:
		return r.DoRetry
	default:
		return DefaultDoRetry
	}
}

func onRetry(retryable Retryable) func(int, time.Duration, error) {
	if r, ok := retryable.(RetryableOnRetry); ok {
		return r.OnRetry
	}

	return func(i int, duration time.Duration, err error) {}
}

func adaptDoRetry(doRetry func(error) bool) func(int, error) (time.Duration, bool) {
	return func(attempt int, err error) (time.Duration, bool) {
		return DefaultDelay, doRetry(err)
	}
}
