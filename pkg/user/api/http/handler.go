package http

import (
	"context"
	"net/http"
	"time"

	"github.com/abialemuel/AI-Proxy-Service/config"
	common "github.com/abialemuel/AI-Proxy-Service/pkg/common/http"
	"github.com/abialemuel/AI-Proxy-Service/pkg/common/http/middleware/authguard"
	"github.com/abialemuel/AI-Proxy-Service/pkg/common/http/validator"
	oauthmanager "github.com/abialemuel/AI-Proxy-Service/pkg/common/oauth"
	"github.com/abialemuel/AI-Proxy-Service/pkg/user/api/http/request"
	"github.com/abialemuel/AI-Proxy-Service/pkg/user/api/http/response"
	"github.com/abialemuel/AI-Proxy-Service/pkg/user/business"
	"github.com/abialemuel/AI-Proxy-Service/pkg/user/business/core"

	"github.com/abialemuel/poly-kit/infrastructure/apm"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	service        business.UserService
	googleOauth    oauthmanager.OAuth2Provider
	microsoftOauth oauthmanager.OAuth2Provider
	config         *config.MainConfig
}

// NewHandler Construct user API handler
func NewHandler(service business.UserService, google oauthmanager.OAuth2Provider, microsoft oauthmanager.OAuth2Provider, cfg *config.MainConfig) *Handler {
	return &Handler{
		service,
		google,
		microsoft,
		cfg,
	}
}

func (h *Handler) GetUser(c echo.Context) error {
	ctx, span := apm.StartTransaction(c.Request().Context(), "Handler::GetUser")
	defer apm.EndTransaction(span)
	// get user id from token
	jwtAtrr := c.Get(authguard.UserAttr).(authguard.JwtClaims)

	tokenInfo := h.service.GetUserTokenUsage(ctx, jwtAtrr.Email)

	// return 200 with user data from jwt
	return c.JSON(http.StatusOK, response.NewUserMeResponse(jwtAtrr, tokenInfo))
}

// GPT4Handler handler for GPT4
func (h *Handler) UserGPT4Handler(c echo.Context) error {
	ctx, span := apm.StartTransaction(c.Request().Context(), "Handler::GPT4UserPrompt")
	defer apm.EndTransaction(span)

	jwtAtrr := c.Get(authguard.UserAttr).(authguard.JwtClaims)
	userID := jwtAtrr.Email

	req := new(request.UserPromptGPTRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, common.NewValidationErrorResponse("Invalid Body"))
	}
	if msg, check := validator.Validation(req); !check {
		return c.JSON(http.StatusBadRequest, common.NewValidationErrorResponse(msg))
	}

	// Create a new context with a longer timeout or no timeout
	longCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	res, err := h.service.UserPromtGPT(longCtx, core.UserPromtGPTRequest{
		UserID:  userID,
		Content: request.ToCoreUserPromptGPTRequest(req.Content),
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, common.NewValidationErrorResponse(err.Error()))
	}

	return c.JSON(http.StatusOK, response.NewUserPromGPTResponse(res))
}

// ServiceGPT4Handler
func (h *Handler) ServiceGPT4Handler(c echo.Context) error {
	ctx, span := apm.StartTransaction(c.Request().Context(), "Handler::GPT4ServicePrompt")
	defer apm.EndTransaction(span)

	req := new(request.ServicePromptGPTRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, common.NewValidationErrorResponse("Invalid Body"))
	}
	if msg, check := validator.Validation(req); !check {
		return c.JSON(http.StatusBadRequest, common.NewValidationErrorResponse(msg))
	}

	// get service from context
	serviceName := c.Get("service").(string)

	// Create a new context with a longer timeout or no timeout
	longCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	res, err := h.service.ServicePrompt(longCtx, core.ServicePromptRequest{
		ServiceName: serviceName,
		Model:       req.Model,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		TopP:        req.TopP,
		Messages:    request.ToCoreMessage(req.Messages),
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, common.NewValidationErrorResponse(err.Error()))
	}

	return c.JSON(http.StatusOK, response.NewServicePromGPTResponse(res))
}

// UserClearContextHandler handler for clearing context
func (h *Handler) UserClearContextHandler(c echo.Context) error {
	ctx, span := apm.StartTransaction(c.Request().Context(), "Handler::ClearContext")
	defer apm.EndTransaction(span)

	jwtAtrr := c.Get(authguard.UserAttr).(authguard.JwtClaims)
	userID := jwtAtrr.Email

	// clear context
	err := h.service.UserClearContext(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, common.NewValidationErrorResponse(err.Error()))
	}

	return c.JSON(http.StatusOK, common.DefaultResponse{Code: 200, Message: "Context Cleared"})
}
