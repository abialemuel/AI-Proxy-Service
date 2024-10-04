package http

import (
	"github.com/abialemuel/AI-Proxy-Service/pkg/common/http/middleware/authguard"

	"github.com/labstack/echo/v4"
)

// RegisterPath Register V1 API path
func RegisterPath(e *echo.Echo, h *Handler, authGuard *authguard.AuthGuard) {
	if h == nil {
		panic("item controller cannot be nil")
	}

	// Auth implementation
	e.GET("v1/auth/login", h.AuthHandler)
	e.GET("v1/auth/google/callback", h.GoogleAuthCallback)
	e.GET("v1/auth/microsoft/callback", h.MicrosoftAuthCallback)
	e.GET("v1/auth/refresh", h.RefreshTokenHandler)

	e.GET("v1/users/me", h.GetUser, authGuard.Bearer)
	e.POST("v1/prompt", h.UserGPT4Handler, authGuard.Bearer)
	e.POST("v1/prompt/new", h.UserClearContextHandler, authGuard.Bearer)

	// Internal service for GPT4
	e.POST("v1/prompt/internal", h.ServiceGPT4Handler, authGuard.Basic)
}
