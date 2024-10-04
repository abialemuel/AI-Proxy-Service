package oauthmanager

import (
	"errors"

	"github.com/abialemuel/AI-Proxy-Service/pkg/common/oauth/google"
	"github.com/abialemuel/AI-Proxy-Service/pkg/common/oauth/microsoft"
	"github.com/abialemuel/AI-Proxy-Service/pkg/common/oauth/model"
)

// ProviderType represents the supported OAuth2 providers.
type ProviderType string

const (
	GoogleProvider    ProviderType = "google"
	MicrosoftProvider ProviderType = "microsoft"
)

// OAuth2Provider defines the interface for OAuth2 operations.
type OAuth2Provider interface {
	GetAuthURL(state string) string
	ExchangeCodeForToken(code string) (*model.TokenResponse, error)
	GetRedirectURL() string
	GetRefreshToken(refreshToken string) (*model.TokenResponse, error)
}

// NewOAuth2Provider creates a new OAuth2 provider adapter based on the given provider type.
func NewOAuth2Provider(providerType ProviderType, tenantID, clientID, clientSecret, redirectURL string) (OAuth2Provider, error) {
	switch providerType {
	case GoogleProvider:
		return google.NewGoogleAdapter(clientID, clientSecret, redirectURL), nil
	case MicrosoftProvider:
		return microsoft.NewMicrosoftAdapter(tenantID, clientID, clientSecret, redirectURL), nil
	default:
		return nil, errors.New("unsupported provider type")
	}
}
