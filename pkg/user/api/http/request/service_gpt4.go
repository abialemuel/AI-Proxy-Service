package request

import "github.com/abialemuel/AI-Proxy-Service/pkg/user/business/core"

type ServicePromptGPTRequest struct {
	Model       string    `json:"model" validate:"required"`
	Temperature float64   `json:"temperature" validate:"required"`
	MaxTokens   int       `json:"max_tokens" validate:"required"`
	TopP        float64   `json:"top_p" validate:"required"`
	Messages    []Message `json:"messages" validate:"required"`
}

type Message struct {
	Role    string    `json:"role" validate:"required"`
	Content []Content `json:"content" validate:"required"`
}

func ToCoreMessage(req []Message) (res []core.MessageRequest) {
	for _, v := range req {
		var message core.MessageRequest
		message.Role = v.Role
		message.Content = ToCoreContent(v.Content)
		res = append(res, message)
	}
	return res
}

func ToCoreContent(req []Content) (res []core.Content) {
	for _, v := range req {
		var content core.Content
		content.Type = v.Type
		if v.Text != nil {
			content.Text = v.Text
		}
		if v.ImageURL != nil {
			content.ImageURL = &core.ImageURL{
				URL: v.ImageURL.URL,
			}
		}
		res = append(res, content)
	}
	return res
}
