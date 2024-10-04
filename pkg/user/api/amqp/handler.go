package amqp

import (
	oauthmanager "github.com/abialemuel/AI-Proxy-Service/pkg/common/oauth"
	"github.com/abialemuel/AI-Proxy-Service/pkg/user/business"
)

type Handler struct {
	service      business.UserService
	oauthManager oauthmanager.OAuth2Provider
}

func NewHandler(s business.UserService) *Handler {
	return &Handler{service: s}
}
