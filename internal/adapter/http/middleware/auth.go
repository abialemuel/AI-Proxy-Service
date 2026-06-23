// Package middleware provides Echo middlewares (auth, recovery, logging).
package middleware

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"

	"github.com/abialemuel/AI-Proxy-Service/internal/adapter/http/handler"
	"github.com/abialemuel/AI-Proxy-Service/internal/domain"
	"github.com/abialemuel/AI-Proxy-Service/internal/pkg/config"
)

// jwtClaims is the in-house claims payload.
type jwtClaims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Tribe string `json:"tribe,omitempty"`
	jwt.RegisteredClaims
}

// Bearer authenticates user requests using a Bearer JWT.
func Bearer(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokStr := extractBearer(c.Request().Header.Get("Authorization"))
			if tokStr == "" {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "missing bearer token"})
			}
			tok, err := jwt.ParseWithClaims(tokStr, &jwtClaims{}, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrTokenSignatureInvalid
				}
				return []byte(secret), nil
			})
			if err != nil || !tok.Valid {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid token"})
			}
			claims, _ := tok.Claims.(*jwtClaims)
			handler.SetPrincipal(c, domain.Principal{
				Kind:    domain.PrincipalUser,
				Subject: claims.Email,
				Tribe:   claims.Tribe,
			})
			return next(c)
		}
	}
}

// Basic authenticates service requests using HTTP Basic against a static list.
func Basic(creds []config.ServiceCreds) echo.MiddlewareFunc {
	index := make(map[string]config.ServiceCreds, len(creds))
	for _, sc := range creds {
		index[sc.Username+":"+sc.Password] = sc
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, pass, ok := c.Request().BasicAuth()
			if !ok {
				c.Response().Header().Set("WWW-Authenticate", `Basic realm="proxy"`)
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "basic auth required"})
			}
			sc, found := index[user+":"+pass]
			if !found {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid service credentials"})
			}
			handler.SetPrincipal(c, domain.Principal{
				Kind:     domain.PrincipalService,
				Subject:  sc.Name,
				Tribe:    sc.Tribe,
				Username: sc.Username,
			})
			return next(c)
		}
	}
}

func extractBearer(h string) string {
	const p = "Bearer "
	if strings.HasPrefix(h, p) {
		return strings.TrimSpace(h[len(p):])
	}
	return ""
}
