package http

import (
	"fmt"
	"net/http"

	common "github.com/abialemuel/AI-Proxy-Service/pkg/common/http"
	"github.com/abialemuel/AI-Proxy-Service/pkg/common/http/validator"
	"github.com/abialemuel/AI-Proxy-Service/pkg/common/oauth/model"
	"github.com/abialemuel/AI-Proxy-Service/pkg/user/api/http/request"
	"github.com/abialemuel/AI-Proxy-Service/pkg/user/api/http/response"

	"github.com/abialemuel/poly-kit/infrastructure/apm"
	"github.com/labstack/echo/v4"
)

const (
	UIPath = "/gpt4/prompts"
)

func (h *Handler) AuthHandler(c echo.Context) (err error) {
	_, span := apm.StartTransaction(c.Request().Context(), "Handler::AuthUser")
	defer apm.EndTransaction(span)

	req := new(request.AuthLogin)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, common.NewValidationErrorResponse("Invalid Body"))
	}
	if msg, check := validator.Validation(req); !check {
		return c.JSON(http.StatusBadRequest, common.NewValidationErrorResponse(msg))
	}

	authUrl := ""
	if req.Provider == "google" {
		authUrl = h.googleOauth.GetAuthURL("google")
	} else if req.Provider == "microsoft" {
		authUrl = h.microsoftOauth.GetAuthURL("microsoft")
	}

	// return json
	return c.JSON(http.StatusOK, response.NewAuthResponse(authUrl))
}

// GoogleAuthCallback Receive Callback
func (h *Handler) GoogleAuthCallback(c echo.Context) error {
	_, span := apm.StartTransaction(c.Request().Context(), "Handler::GoogleCallback")
	defer apm.EndTransaction(span)

	req := new(request.AuthCallback)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, common.NewValidationErrorResponse("Invalid Body"))
	}
	if msg, check := validator.Validation(req); !check {
		fmt.Println(msg)
		return c.JSON(http.StatusBadRequest, common.NewValidationErrorResponse(msg))
	}

	token, err := h.googleOauth.ExchangeCodeForToken(req.Code)
	if err != nil {
		return c.JSON(http.StatusBadRequest, common.NewValidationErrorResponse(err.Error()))
	}

	// redirect to url
	return c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s?access_token=%s&refresh_token=%s", h.config.UI.Host+UIPath, token.IDToken, token.RefreshToken))
}

// MicrosoftAuthCallback Receive Callback
func (h *Handler) MicrosoftAuthCallback(c echo.Context) error {
	_, span := apm.StartTransaction(c.Request().Context(), "Handler::MicrosoftCallback")
	defer apm.EndTransaction(span)

	req := new(request.AuthCallback)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, common.NewValidationErrorResponse("Invalid Body"))
	}
	if msg, check := validator.Validation(req); !check {
		return c.JSON(http.StatusBadRequest, common.NewValidationErrorResponse(msg))
	}

	token, err := h.microsoftOauth.ExchangeCodeForToken(req.Code)
	if err != nil {
		return c.JSON(http.StatusBadRequest, common.NewValidationErrorResponse(err.Error()))
	}

	// redirect to url with token
	return c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s?access_token=%s&refresh_token=%s", h.config.UI.Host+UIPath, token.IDToken, token.RefreshToken))
}

// RefreshTokenHandler Refresh Token
func (h *Handler) RefreshTokenHandler(c echo.Context) error {
	_, span := apm.StartTransaction(c.Request().Context(), "Handler::RefreshToken")
	defer apm.EndTransaction(span)

	req := new(request.AuthRefresh)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, common.NewValidationErrorResponse("Invalid Body"))
	}
	if msg, check := validator.Validation(req); !check {
		return c.JSON(http.StatusBadRequest, common.NewValidationErrorResponse(msg))
	}

	token := new(model.TokenResponse)
	var err error
	if req.Provider == "google" {
		token, err = h.googleOauth.GetRefreshToken(req.RefreshToken)
	} else if req.Provider == "microsoft" {
		token, err = h.microsoftOauth.GetRefreshToken(req.RefreshToken)
	}
	if err != nil {
		return c.JSON(http.StatusBadRequest, common.NewValidationErrorResponse(err.Error()))
	}

	// return json
	return c.JSON(http.StatusOK, response.NewTokenResponse(token.IDToken, token.RefreshToken))
}
