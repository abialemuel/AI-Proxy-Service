// Package handler holds HTTP handlers. They are intentionally thin:
// decode → validate → call usecase → translate → write.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/abialemuel/AI-Proxy-Service/internal/adapter/http/dto"
	"github.com/abialemuel/AI-Proxy-Service/internal/usecase"
)

const requestTimeout = 90 * time.Second

// ChatHandler exposes the chat use case over HTTP.
type ChatHandler struct {
	chat *usecase.ChatService
}

func NewChatHandler(c *usecase.ChatService) *ChatHandler {
	return &ChatHandler{chat: c}
}

// Register attaches routes to the given echo group.
func (h *ChatHandler) Register(g *echo.Group, userMW, serviceMW echo.MiddlewareFunc) {
	g.POST("/prompt", h.UserPrompt, userMW)
	g.POST("/prompt/new", h.ClearContext, userMW)
	g.POST("/prompt/internal", h.ServicePrompt, serviceMW)
	g.GET("/providers", h.ListProviders, userMW)
}

// UserPrompt handles authenticated user chat (stateful).
func (h *ChatHandler) UserPrompt(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()

	var req dto.ChatRequest
	if err := c.Bind(&req); err != nil {
		return writeError(c, http.StatusBadRequest, "invalid request body")
	}
	userID := principalSubject(c)
	if userID == "" {
		return writeError(c, http.StatusUnauthorized, "missing user identity")
	}

	resp, err := h.chat.UserChat(ctx, usecase.UserChatInput{
		UserID:  userID,
		Request: req.ToDomain(),
	})
	if err != nil {
		return writeAppError(c, err)
	}
	return c.JSON(http.StatusOK, dto.FromDomain(resp))
}

// ServicePrompt handles backend-service chat (stateless).
func (h *ChatHandler) ServicePrompt(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
	defer cancel()

	var req dto.ChatRequest
	if err := c.Bind(&req); err != nil {
		return writeError(c, http.StatusBadRequest, "invalid request body")
	}
	resp, err := h.chat.ServiceChat(ctx, req.ToDomain())
	if err != nil {
		return writeAppError(c, err)
	}
	return c.JSON(http.StatusOK, dto.FromDomain(resp))
}

// ClearContext drops the caller's conversation history.
func (h *ChatHandler) ClearContext(c echo.Context) error {
	userID := principalSubject(c)
	if userID == "" {
		return writeError(c, http.StatusUnauthorized, "missing user identity")
	}
	if err := h.chat.ClearContext(c.Request().Context(), userID); err != nil {
		return writeAppError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ListProviders returns the registered LLM providers.
func (h *ChatHandler) ListProviders(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{"providers": h.chat.AvailableProviders()})
}
