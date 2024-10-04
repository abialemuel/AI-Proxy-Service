package response

import "github.com/abialemuel/AI-Proxy-Service/pkg/user/business/core"

type ServicePromGPT struct {
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	Usage       Usage   `json:"usage"`
	Content     string  `json:"content"`
}

type Usage struct {
	CompletionTokens int `json:"completion_tokens"`
	PromptTokens     int `json:"prompt_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
type ServicePromGPTResponse struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Payload ServicePromGPT `json:"payload"`
}

func NewServicePromGPTResponse(v core.ServicePromGPTResponse) *ServicePromGPTResponse {
	var ResultResponse ServicePromGPTResponse
	payload := ServicePromGPT{
		Model:   v.GPT4PromptResponse.Model,
		Content: v.GPT4PromptResponse.Choices[0].Message.Content,
		Usage: Usage{
			CompletionTokens: v.GPT4PromptResponse.Usage.CompletionTokens,
			PromptTokens:     v.GPT4PromptResponse.Usage.PromptTokens,
			TotalTokens:      v.GPT4PromptResponse.Usage.TotalTokens,
		},
	}

	ResultResponse.Code = 200
	ResultResponse.Message = "Success"
	ResultResponse.Payload = payload
	return &ResultResponse
}
