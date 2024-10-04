package core

import "github.com/abialemuel/AI-Proxy-Service/pkg/user/modules/gpt4_webservice"

type ServicePromptRequest struct {
	Model       string           `json:"model"`
	ServiceName string           `json:"service_name"`
	Temperature float64          `json:"temperature"`
	MaxTokens   int              `json:"max_tokens"`
	TopP        float64          `json:"top_p"`
	Messages    []MessageRequest `json:"messages"`
}

type MessageRequest struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

func ToWebServicePromtGPTMsgRequest(req []MessageRequest) (res []gpt4_webservice.MessageReq) {
	for _, v := range req {
		var message gpt4_webservice.MessageReq
		message.Role = v.Role
		message.Content = ToWebServiceUserPromtGPTContentRequest(v.Content)
		res = append(res, message)
	}
	return res
}

type ServicePromGPTResponse struct {
	GPT4PromptResponse
	UserID string `json:"user_id"`
}
