package google

import (
	"context"

	"github.com/abialemuel/AI-Proxy-Service/pkg/common/oauth/model"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleAdapter is an adapter for Google OAuth2.
type GoogleAdapter struct {
	config *oauth2.Config
}

// NewGoogleAdapter creates a new GoogleAdapter with the given client configuration.
func NewGoogleAdapter(clientID, clientSecret, redirectURL string) *GoogleAdapter {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint:     google.Endpoint,
	}
	return &GoogleAdapter{config: config}
}

// GetAuthURL generates the Google OAuth2 authorization URL.
func (g *GoogleAdapter) GetAuthURL(state string) string {
	return g.config.AuthCodeURL(state)
}

// Get Refresh Token
func (m *GoogleAdapter) GetRefreshToken(refreshToken string) (*model.TokenResponse, error) {
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

// ExchangeCodeForToken exchanges an authorization code for tokens.
func (g *GoogleAdapter) ExchangeCodeForToken(code string) (*model.TokenResponse, error) {
	token, err := g.config.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, err
	}
	idToken := token.Extra("id_token").(string)
	return &model.TokenResponse{AccessToken: token.AccessToken, RefreshToken: token.RefreshToken, IDToken: idToken}, nil
}

// GetRedirectURL returns the redirect URL configured for the Google OAuth2 provider.
func (g *GoogleAdapter) GetRedirectURL() string {
	return g.config.RedirectURL
}
