package handler

import (
	"github.com/labstack/echo/v4"

	"github.com/abialemuel/AI-Proxy-Service/internal/domain"
)

// ContextKey is the key used to stash the authenticated principal in echo.Context.
const ContextKey = "principal"

// SetPrincipal stores the principal on the request context.
func SetPrincipal(c echo.Context, p domain.Principal) {
	c.Set(ContextKey, p)
}

// PrincipalOf retrieves the principal previously stashed by an auth middleware.
func PrincipalOf(c echo.Context) (domain.Principal, bool) {
	v := c.Get(ContextKey)
	if v == nil {
		return domain.Principal{}, false
	}
	p, ok := v.(domain.Principal)
	return p, ok
}

func principalSubject(c echo.Context) string {
	if p, ok := PrincipalOf(c); ok {
		return p.Subject
	}
	return ""
}
