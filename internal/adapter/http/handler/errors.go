package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	apierrors "github.com/abialemuel/AI-Proxy-Service/internal/pkg/errors"
)

// errorBody is the canonical error envelope returned to clients.
type errorBody struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(c echo.Context, status int, msg string) error {
	return c.JSON(status, errorBody{
		Error:   http.StatusText(status),
		Code:    string(apierrors.CodeOf(apierrors.New(apierrors.CodeValidation, msg))),
		Message: msg,
	})
}

// writeAppError maps a domain/app error to a status code and JSON envelope.
func writeAppError(c echo.Context, err error) error {
	code := apierrors.CodeOf(err)
	status := statusFor(code)
	return c.JSON(status, errorBody{
		Error:   http.StatusText(status),
		Code:    string(code),
		Message: err.Error(),
	})
}

func statusFor(code apierrors.Code) int {
	switch code {
	case apierrors.CodeValidation:
		return http.StatusBadRequest
	case apierrors.CodeUnauthorized:
		return http.StatusUnauthorized
	case apierrors.CodeForbidden:
		return http.StatusForbidden
	case apierrors.CodeNotFound:
		return http.StatusNotFound
	case apierrors.CodeConflict:
		return http.StatusConflict
	case apierrors.CodeRateLimited, apierrors.CodeQuotaExceeded:
		return http.StatusTooManyRequests
	case apierrors.CodeProviderTimeout:
		return http.StatusGatewayTimeout
	case apierrors.CodeProviderBadReply:
		return http.StatusBadGateway
	case apierrors.CodeUnsupportedModel:
		return http.StatusUnprocessableEntity
	case apierrors.CodeProviderError:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}
