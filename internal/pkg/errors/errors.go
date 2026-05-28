// Package errors defines the domain error taxonomy used across the service.
// Adapters wrap vendor errors into these; the HTTP edge maps them to status codes.
package errors

import (
	"errors"
	"fmt"
)

// Code is a stable, machine-readable identifier for an error class.
type Code string

const (
	CodeUnknown          Code = "unknown"
	CodeValidation       Code = "validation"
	CodeUnauthorized     Code = "unauthorized"
	CodeForbidden        Code = "forbidden"
	CodeNotFound         Code = "not_found"
	CodeConflict         Code = "conflict"
	CodeRateLimited      Code = "rate_limited"
	CodeProviderTimeout  Code = "provider_timeout"
	CodeProviderBadReply Code = "provider_bad_reply"
	CodeProviderError    Code = "provider_error"
	CodeQuotaExceeded    Code = "quota_exceeded"
	CodeUnsupportedModel Code = "unsupported_model"
)

// Error is a typed, wrappable application error.
type Error struct {
	Code    Code
	Message string
	cause   error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap supports errors.Is / errors.As.
func (e *Error) Unwrap() error { return e.cause }

// New constructs a fresh Error.
func New(code Code, msg string) *Error {
	return &Error{Code: code, Message: msg}
}

// Wrap wraps a cause with a code and message.
func Wrap(code Code, msg string, cause error) *Error {
	return &Error{Code: code, Message: msg, cause: cause}
}

// CodeOf extracts the Code from any error chain, defaulting to CodeUnknown.
func CodeOf(err error) Code {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return CodeUnknown
}
