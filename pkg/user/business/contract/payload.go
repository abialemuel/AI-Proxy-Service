package contract

import (
	oauthmanager "github.com/abialemuel/AI-Proxy-Service/pkg/common/oauth"
)

type AuthPayload struct {
	AuthType *oauthmanager.ProviderType
}
