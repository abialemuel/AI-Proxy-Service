// Package http wires the echo server.
package http

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"github.com/abialemuel/AI-Proxy-Service/internal/adapter/http/handler"
	"github.com/abialemuel/AI-Proxy-Service/internal/adapter/http/middleware"
	"github.com/abialemuel/AI-Proxy-Service/internal/pkg/config"
)

// Server bundles the echo instance and exposed routes.
type Server struct {
	E   *echo.Echo
	cfg *config.Config
}

// New constructs the echo server with common middleware and routes.
func New(cfg *config.Config, chat *handler.ChatHandler) *Server {
	e := echo.New()
	e.HideBanner = true
	e.Server.ReadTimeout = secs(cfg.HTTP.ReadTimeoutSec, 15)
	e.Server.WriteTimeout = secs(cfg.HTTP.WriteTimeoutSec, 120)
	e.Server.IdleTimeout = secs(cfg.HTTP.IdleTimeoutSec, 60)

	e.Use(echomw.Recover())
	e.Use(echomw.RequestID())
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderContentType, echo.HeaderAuthorization},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete},
	}))

	e.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.GET("/health", func(c echo.Context) error { return c.String(http.StatusOK, "OK") })
	e.GET("/ready", func(c echo.Context) error { return c.String(http.StatusOK, "READY") })

	v1 := e.Group("/v1")
	chat.Register(v1, middleware.Bearer(cfg.JWT.Secret), middleware.Basic(cfg.Services))

	return &Server{E: e, cfg: cfg}
}

func secs(v, def int) time.Duration {
	if v <= 0 {
		v = def
	}
	return time.Duration(v) * time.Second
}
