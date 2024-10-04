package response

import (
	"time"

	"github.com/abialemuel/AI-Proxy-Service/pkg/common/http/middleware/authguard"
	"github.com/abialemuel/AI-Proxy-Service/pkg/user/business/core"
)

type UserPromGPT struct {
	Content string `json:"content"`
}

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type TokenResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Payload Token
}

type UserAuth struct {
	AuthURL string `json:"auth_url"`
}

type UserAuthResponse struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Payload UserAuth `json:"payload"`
}

type UserPromGPTResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Payload UserPromGPT `json:"payload"`
}

type UserMeResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Payload UserMe `json:"payload"`
}

type UserMe struct {
	Issuer     string    `json:"issuer"`
	UserID     string    `json:"user_id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	Picture    string    `json:"picture"`
	ExpiresAt  time.Time `json:"expires_at"`
	TokenLimit int       `json:"token_limit"`
	TokenUsage int       `json:"token_usage"`
	Warning    bool      `json:"warning"`
}

func NewUserMeResponse(v authguard.JwtClaims, tokenInfo core.UserTokenUsage) *UserMeResponse {
	var ResultResponse UserMeResponse
	payload := UserMe{
		Issuer:     v.Issuer,
		UserID:     v.Subject,
		Email:      v.Email,
		Name:       v.Name,
		Picture:    v.Picture,
		ExpiresAt:  time.Unix(v.ExpiresAt.Unix(), 0),
		TokenLimit: tokenInfo.TokenLimit,
		TokenUsage: tokenInfo.TokenUsage,
		Warning:    tokenInfo.Warning,
	}

	ResultResponse.Code = 200
	ResultResponse.Message = "Success"
	ResultResponse.Payload = payload
	return &ResultResponse
}

func NewUserPromGPTResponse(v core.UserPromGPTResponse) *UserPromGPTResponse {
	var ResultResponse UserPromGPTResponse
	payload := UserPromGPT{
		Content: v.GPT4PromptResponse.Choices[0].Message.Content,
	}

	ResultResponse.Code = 200
	ResultResponse.Message = "Success"
	ResultResponse.Payload = payload
	return &ResultResponse
}

func NewAuthResponse(authUrl string) *UserAuthResponse {
	var ResultResponse UserAuthResponse
	payload := UserAuth{
		AuthURL: authUrl,
	}

	ResultResponse.Code = 200
	ResultResponse.Message = "Success"
	ResultResponse.Payload = payload
	return &ResultResponse
}

func NewTokenResponse(token string, refreshToken string) *TokenResponse {
	var ResultResponse TokenResponse
	payload := Token{
		AccessToken:  token,
		RefreshToken: refreshToken,
	}

	ResultResponse.Code = 200
	ResultResponse.Message = "Success"
	ResultResponse.Payload = payload
	return &ResultResponse
}
