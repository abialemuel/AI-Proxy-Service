package microsoft

import (
	"context"

	"github.com/abialemuel/AI-Proxy-Service/pkg/common/oauth/model"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

// MicrosoftAdapter is an adapter for Microsoft OAuth2.
type MicrosoftAdapter struct {
	config *oauth2.Config
}

// NewMicrosoftAdapter creates a new MicrosoftAdapter with the given client configuration.
func NewMicrosoftAdapter(tenantID, clientID, clientSecret, redirectURL string) *MicrosoftAdapter {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "profile", "email", "offline_access"},
		Endpoint:     microsoft.AzureADEndpoint(tenantID),
	}
	return &MicrosoftAdapter{config: config}
}

// GetAuthURL generates the Microsoft OAuth2 authorization URL.
func (m *MicrosoftAdapter) GetAuthURL(state string) string {
	return m.config.AuthCodeURL(state)
}

// ExchangeCodeForToken exchanges an authorization code for tokens.
func (m *MicrosoftAdapter) ExchangeCodeForToken(code string) (*model.TokenResponse, error) {
	token, err := m.config.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, err
	}
	idToken := token.Extra("id_token").(string)

	return &model.TokenResponse{AccessToken: token.AccessToken, RefreshToken: token.RefreshToken, IDToken: idToken}, nil
}

// Get Refresh Token
func (m *MicrosoftAdapter) GetRefreshToken(refreshToken string) (*model.TokenResponse, error) {
	oldToken := &oauth2.Token{
		RefreshToken: refreshToken,
	}
	tokenSource := m.config.TokenSource(context.Background(), oldToken)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, err
	}
	idToken := newToken.Extra("id_token").(string)

	return &model.TokenResponse{AccessToken: newToken.AccessToken, RefreshToken: newToken.RefreshToken, IDToken: idToken}, nil
}

// GetRedirectURL returns the redirect URL configured for the Microsoft OAuth2 provider.
func (m *MicrosoftAdapter) GetRedirectURL() string {
	return m.config.RedirectURL
}
