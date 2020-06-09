package main

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

type ResponseError struct {
	Status  int
	Message string
	Errors  []string
}

type ResponseDecodingError struct {
	ResponseError
	Body []byte
}

type RateLimitError struct {
	ResponseError
	RetryAfter time.Duration
}

func (e ResponseError) Error() string {
	const (
		unknown = "unknown error"
		errSep  = ", "
		msgFmt  = "%03d: %s"
	)

	msg := e.Message
	if msg == "" {
		msg = strings.Join(e.Errors, errSep)
	}
	if msg == "" && e.Status > 0 {
		msg = http.StatusText(e.Status)
	}
	if msg == "" {
		msg = unknown
	}

	if e.Status > 0 {
		return fmt.Sprintf(msgFmt, e.Status, msg)
	}

	return msg
}

func coerceError(v interface{}) string {
	const sep = ", "

	switch v := v.(type) {
	case string:
		return v
	case []interface{}:
		s := coerceErrorSlice(v)
		return strings.Join(s, sep)
	case []string:
		return strings.Join(v, sep)
	case map[string]interface{}:
		s := coerceErrorMap(v)
		return strings.Join(s, sep)
	default:
		return fmt.Sprint(v)
	}
}

func coerceErrorSlice(v []interface{}) []string {
	rv := make([]string, len(v))
	for idx, v := range v {
		rv[idx] = coerceError(v)
	}
	return rv
}

func coerceErrorMap(v map[string]interface{}) []string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	rv := make([]string, len(keys))
	for idx, k := range keys {
		v := v[k]
		rv[idx] = fmt.Sprintf("%s: %s", k, coerceError(v))
	}
	return rv
}

func (e *ResponseError) setErrors(errors interface{}) {
	switch errors := errors.(type) {
	case nil:
		return
	case string:
		e.Message = errors
	case []interface{}:
		e.Errors = coerceErrorSlice(errors)
	case map[string]interface{}:
		e.Errors = coerceErrorMap(errors)
	default:
		if e.Message == "" {
			e.Message = fmt.Sprint(errors)
		} else {
			e.Message = fmt.Sprintf("%s: %v", e.Message, errors)
		}
	}
}

func NewResponseDecodingError(res *http.Response, err error, data []byte) error {
	return ResponseDecodingError{
		ResponseError: ResponseError{
			Status:  res.StatusCode,
			Message: err.Error(),
		},
		Body: data,
	}
}
