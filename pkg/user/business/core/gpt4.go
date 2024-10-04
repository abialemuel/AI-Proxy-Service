package core

import "github.com/abialemuel/AI-Proxy-Service/pkg/user/modules/gpt4_webservice"

// type GPT4PromptRequest struct {
// 	Model       string    `json:"model"`
// 	Messages    []Message `json:"messages"`
// 	Temperature float64   `json:"temperature"`
// 	MaxTokens   int       `json:"max_tokens"`
// 	TopP        float64   `json:"top_p"`
// }

type GPT4PromptResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created float64   `json:"created"`
	Model   string    `json:"model"`
	Choices []Choices `json:"choices"`
	Usage   Usage     `json:"usage"`
}

type Message struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

type Choices struct {
	Index   int     `json:"index"`
	Message Message `json:"message"`
}

type Usage struct {
	CompletionTokens int `json:"completion_tokens"`
	PromptTokens     int `json:"prompt_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func ToCoreGPT4PromptResponse(p gpt4_webservice.GPT4PromptResponseDao) GPT4PromptResponse {
	return GPT4PromptResponse{
		ID:      p.ID,
		Object:  p.Object,
		Created: p.Created,
		Model:   p.Model,
		Choices: ToCoreChoices(p.Choices),
		Usage:   ToCoreUsage(p.Usage),
	}
}

func ToCoreChoices(c []gpt4_webservice.Choices) []Choices {
	var coreChoices []Choices
	for _, choice := range c {
		coreChoices = append(coreChoices, Choices{
			Index: choice.Index,
			Message: Message{
				Content: choice.Message.Content,
				Role:    choice.Message.Role,
			},
		})
	}
	return coreChoices
}

func ToCoreUsage(u gpt4_webservice.Usage) Usage {
	return Usage{
		CompletionTokens: u.CompletionTokens,
		PromptTokens:     u.PromptTokens,
		TotalTokens:      u.TotalTokens,
	}
}
