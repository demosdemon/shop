package retry

import (
	"time"
)

type Option func(retryable *funcRetryable)

func New(do func() error, options ...Option) Retryable {
	r := funcRetryable{
		do:          do,
		doRetry:     func(attempts int, err error) (time.Duration, bool) { return DefaultDelay, true },
		maxAttempts: DefaultMaxAttempts,
	}

	for _, opt := range options {
		opt(&r)
	}

	return r
}

func WithMaxAttempts(maxAttempts int) Option {
	return func(retryable *funcRetryable) {
		retryable.maxAttempts = maxAttempts
	}
}

func WithDoRetry(doRetry func(error) bool) Option {
	return func(retryable *funcRetryable) {
		retryable.doRetry = adaptDoRetry(doRetry)
	}
}

func WithDoRetryWithDelay(doRetry func(int, error) (time.Duration, bool)) Option {
	return func(retryable *funcRetryable) {
		retryable.doRetry = doRetry
	}
}

func WithDelay(delay time.Duration) Option {
	return func(retryable *funcRetryable) {
		retryable.doRetry = func(attempt int, err error) (time.Duration, bool) {
			return delay, true
		}
	}
}

func WithOnRetry(onRetry func(int, time.Duration, error)) Option {
	return func(retryable *funcRetryable) {
		retryable.onRetry = onRetry
	}
}

type funcRetryable struct {
	do          func() error
	doRetry     func(int, error) (time.Duration, bool)
	onRetry     func(int, time.Duration, error)
	maxAttempts int
}

func (r funcRetryable) Do() error {
	return r.do()
}

func (r funcRetryable) DoRetry(attempt int, err error) (time.Duration, bool) {
	return r.doRetry(attempt, err)
}

func (r funcRetryable) MaxAttempts() int {
	return r.maxAttempts
}

func (r funcRetryable) OnRetry(attempt int, wait time.Duration, err error) {
	f := r.onRetry
	if f != nil {
		f(attempt, wait, err)
	}
}
