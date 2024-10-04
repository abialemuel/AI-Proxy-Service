package authguard

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	common "github.com/abialemuel/AI-Proxy-Service/pkg/common/http"

	"github.com/abialemuel/AI-Proxy-Service/config"
	goJwt "github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

// Constants for handling JWT and keys
var (
	UserAttr          = "userAttr"
	PrefixHeader      = "Bearer "
	PrefixHeaderBasic = "Basic "
	googleIssuer      = "https://accounts.google.com"
	microsoftIssuer   = "https://login.microsoftonline.com/%s/v2.0"
	googleCertsURL    = "https://www.googleapis.com/oauth2/v1/certs"
	microsoftCertsURL = "https://login.microsoftonline.com/common/discovery/keys"
)

type JWK struct {
	Kty string   `json:"kty"`
	Use string   `json:"use"`
	Kid string   `json:"kid"`
	X5c []string `json:"x5c"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JwtClaims represents the claims extracted from the JWT
type JwtClaims struct {
	Picture string `json:"picture"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	goJwt.RegisteredClaims
}

type BasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthGuard holds dependencies like API key and configuration
type AuthGuard struct {
	cfg       config.MainConfig
	certs     map[string]*rsa.PublicKey
	certsLock sync.RWMutex
	services  map[string]BasicAuth
}

// NewAuthGuard creates a new instance of AuthGuard
func NewAuthGuard(cfg config.MainConfig) *AuthGuard {
	return &AuthGuard{
		cfg:   cfg,
		certs: make(map[string]*rsa.PublicKey),
	}
}

// Add AuthGuard.services
func (g *AuthGuard) AddService(services []config.BackendService) {
	g.services = make(map[string]BasicAuth)
	for _, service := range services {
		g.services[service.Name] = BasicAuth{
			Username: service.Username,
			Password: service.Password,
		}
	}
}

// Bearer middleware validates JWT tokens, handling multiple OAuth2 providers
func (g *AuthGuard) Bearer(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")

		if !strings.HasPrefix(authHeader, PrefixHeader) {
			return c.JSON(http.StatusUnauthorized, common.NewUnauthorizedResponse("Authorization header missing/invalid"))
		}

		token := strings.TrimPrefix(authHeader, PrefixHeader)
		claims, err := g.ParseAndVerify(token)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, common.NewUnauthorizedResponse(err.Error()))
		}

		c.Set(UserAttr, claims)

		return next(c)
	}
}

// Basic middleware validates Basic Auth tokens
func (g *AuthGuard) Basic(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		serviceHeader := c.Request().Header.Get("X-Service")

		if !strings.HasPrefix(authHeader, PrefixHeaderBasic) {
			return c.JSON(http.StatusUnauthorized, common.NewUnauthorizedResponse("Authorization header missing/invalid"))
		}

		// validate if service is allowed
		if _, ok := g.services[serviceHeader]; !ok {
			return c.JSON(http.StatusUnauthorized, common.NewUnauthorizedResponse("Service not allowed"))
		}

		username, password, ok := c.Request().BasicAuth()
		if !ok {
			return c.JSON(http.StatusUnauthorized, common.NewUnauthorizedResponse("Invalid token"))
		}

		// compare with the stored credentials
		if username != g.services[serviceHeader].Username || password != g.services[serviceHeader].Password {
			return c.JSON(http.StatusUnauthorized, common.NewUnauthorizedResponse("Invalid Basic Auth token"))
		}

		// set the service name to the context
		c.Set("service", serviceHeader)

		return next(c)
	}
}

// ParseAndVerify handles JWT parsing and verification for multiple providers
func (g *AuthGuard) ParseAndVerify(accessToken string) (JwtClaims, error) {
	// Check if the token is a JWT
	if strings.Count(accessToken, ".") == 2 {
		return g.verifyJWT(accessToken)
	} else {
		// unsupported token type, return error
		return JwtClaims{}, errors.New("unsupported token type")
	}
}

// verifyJWT verifies the JWT token locally
func (g *AuthGuard) verifyJWT(accessToken string) (JwtClaims, error) {
	token, err := goJwt.ParseWithClaims(accessToken, &JwtClaims{}, func(token *goJwt.Token) (interface{}, error) {
		claims, ok := token.Claims.(*JwtClaims)
		if !ok {
			return nil, errors.New("invalid token claims")
		}

		// Ensure the token is signed with RSA (RS256)
		if _, ok := token.Method.(*goJwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Determine the provider by the `iss` claim
		switch claims.Issuer {
		case googleIssuer:
			return g.getGooglePublicKey(token.Header["kid"].(string))
		case fmt.Sprintf(microsoftIssuer, g.cfg.MicrosoftOauth.TenantID):
			return g.getMicrosoftPublicKey(token.Header["kid"].(string))
		default:
			return nil, fmt.Errorf("issuer not recognized: %s", claims.Issuer)
		}
	})

	if err != nil || !token.Valid {
		return JwtClaims{}, fmt.Errorf("JWT verification failed: %v. Please re-login", err)
	}

	claims, ok := token.Claims.(*JwtClaims)
	if !ok {
		return JwtClaims{}, errors.New("failed to extract claims")
	}

	// Verify audience
	if !g.verifyAudience(claims.Issuer, claims.Audience) {
		return JwtClaims{}, errors.New("invalid audience")
	}

	return *claims, nil
}

// verifyAudience checks if the audience claim matches the expected audience
func (g *AuthGuard) verifyAudience(issuer string, aud []string) bool {
	switch issuer {
	case googleIssuer:
		return aud[0] == g.cfg.GoogleOauth.ClientID
	case fmt.Sprintf(microsoftIssuer, g.cfg.MicrosoftOauth.TenantID):
		return aud[0] == g.cfg.MicrosoftOauth.ClientID
	}

	return false
}

// getMicrosoftPublicKey fetches the public key from Microsoft's JWKS URL
func (g *AuthGuard) getMicrosoftPublicKey(kid string) (*rsa.PublicKey, error) {
	g.certsLock.RLock()
	key, exists := g.certs[kid]
	g.certsLock.RUnlock()

	if exists {
		return key, nil
	}

	resp, err := http.Get(microsoftCertsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Microsoft JWKS: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch Microsoft JWKS: %s", resp.Status)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode Microsoft JWKS: %v", err)
	}

	g.certsLock.Lock()
	defer g.certsLock.Unlock()

	for _, jwk := range jwks.Keys {
		if jwk.Kid == kid {
			certPEM := "-----BEGIN CERTIFICATE-----\n" + jwk.X5c[0] + "\n-----END CERTIFICATE-----"
			pubKey, err := parseRSAPublicKeyFromPEM([]byte(certPEM))
			if err != nil {
				return nil, fmt.Errorf("failed to parse RSA public key: %v", err)
			}
			g.certs[kid] = pubKey
			return pubKey, nil
		}
	}

	return nil, fmt.Errorf("public key not found for kid: %s", kid)
}

// getGooglePublicKey fetches the public key from Google's certs URL
func (g *AuthGuard) getGooglePublicKey(kid string) (*rsa.PublicKey, error) {
	g.certsLock.RLock()
	key, exists := g.certs[kid]
	g.certsLock.RUnlock()

	if exists {
		return key, nil
	}

	resp, err := http.Get(googleCertsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Google certs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch Google certs: %s", resp.Status)
	}

	var certs map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&certs); err != nil {
		return nil, fmt.Errorf("failed to decode Google certs: %v", err)
	}

	g.certsLock.Lock()
	defer g.certsLock.Unlock()

	for k, v := range certs {
		pubKey, err := parseRSAPublicKeyFromPEM([]byte(v))
		if err != nil {
			return nil, fmt.Errorf("failed to parse RSA public key: %v", err)
		}
		g.certs[k] = pubKey
	}

	key, exists = g.certs[kid]
	if !exists {
		return nil, fmt.Errorf("public key not found for kid: %s", kid)
	}

	return key, nil
}

// parseRSAPublicKeyFromPEM parses an RSA public key from PEM encoded data
func parseRSAPublicKeyFromPEM(pemData []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("failed to decode PEM block containing certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %v", err)
	}

	rsaPub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}

	return rsaPub, nil
}
