package core

import (
	"github.com/abialemuel/AI-Proxy-Service/pkg/user/modules/gpt4_webservice"
	"github.com/abialemuel/AI-Proxy-Service/pkg/user/modules/repository"
)

type UserPromtGPTRequest struct {
	Content []Content `json:"content"`
	UserID  string    `json:"user_id"`
}

type UserPromGPTResponse struct {
	GPT4PromptResponse
	UserID string `json:"user_id"`
}

type UserTokenUsage struct {
	TokenLimit int  `json:"token_limit"`
	TokenUsage int  `json:"token_usage"`
	Warning    bool `json:"warning"`
}

type Content struct {
	Type     string    `json:"type" validate:"required"`
	Text     *string   `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

func ToWebServiceUserPromtGPTContentRequest(req []Content) (res []gpt4_webservice.Content) {
	for _, v := range req {
		var content gpt4_webservice.Content
		content.Type = v.Type
		if v.Text != nil {
			content.Text = v.Text
		}
		if v.ImageURL != nil {
			content.ImageURL = &gpt4_webservice.ImageURL{
				URL: v.ImageURL.URL,
			}
		}
		res = append(res, content)
	}
	return res
}

func ToContentRepo(req []Content) (res []repository.Content) {
	for _, v := range req {
		var content repository.Content
		content.Type = v.Type
		if v.Text != nil {
			content.Text = v.Text
		}
		if v.ImageURL != nil {
			content.ImageURL = v.ImageURL.URL
		}
		res = append(res, content)
	}
	return res
}
