package platform

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// Error is a command failure with an HTTP status and a message safe to show to clients.
type Error struct {
	Status  int
	Message string
	// Err is the underlying cause; it is logged, never sent to clients.
	Err error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// InvalidParam reports a bad parameter value (400).
func InvalidParam(name, format string, args ...any) error {
	return &Error{Status: http.StatusBadRequest, Message: fmt.Sprintf("invalid parameter %q: %s", name, fmt.Sprintf(format, args...))}
}

// MissingCredential reports an unconfigured provider credential (401).
func MissingCredential(provider, key string) error {
	return &Error{Status: http.StatusUnauthorized, Message: fmt.Sprintf("missing credential: %s.%s", provider, key)}
}

// NotFound reports that the upstream has no data for the request (404).
func NotFound(format string, args ...any) error {
	return &Error{Status: http.StatusNotFound, Message: fmt.Sprintf(format, args...)}
}

// Upstream reports a failing data provider (502).
func Upstream(provider string, err error) error {
	return &Error{Status: http.StatusBadGateway, Message: fmt.Sprintf("upstream %s unavailable", provider), Err: err}
}

// statusOf maps any error to an HTTP status and a client-safe message.
func statusOf(err error) (int, string) {
	var e *Error
	if errors.As(err, &e) {
		return e.Status, e.Message
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return http.StatusGatewayTimeout, "command timed out"
	}
	return http.StatusInternalServerError, "internal error"
}
