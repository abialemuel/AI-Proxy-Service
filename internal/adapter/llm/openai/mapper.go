package openai

import (
	"time"

	"github.com/abialemuel/AI-Proxy-Service/internal/domain"
	"github.com/abialemuel/AI-Proxy-Service/pkg/openaicompat"
)

func toOpenAIRequest(r domain.ChatRequest) openaicompat.ChatRequest {
	out := openaicompat.ChatRequest{
		Model:       r.Model,
		Temperature: r.Temperature,
		TopP:        r.TopP,
		MaxTokens:   r.MaxTokens,
		Stream:      r.Stream,
	}
	out.Messages = make([]openaicompat.Message, 0, len(r.Messages))
	for _, m := range r.Messages {
		out.Messages = append(out.Messages, openaicompat.Message{
			Role:    string(m.Role),
			Content: toOpenAIParts(m.Parts),
		})
	}
	return out
}

func toOpenAIParts(parts []domain.ContentPart) []openaicompat.ContentPart {
	out := make([]openaicompat.ContentPart, 0, len(parts))
	for _, p := range parts {
		switch p.Kind {
		case domain.PartImageURL:
			out = append(out, openaicompat.ContentPart{
				Type:     "image_url",
				ImageURL: &openaicompat.ImageURL{URL: p.ImageURL},
			})
		default:
			txt := p.Text
			out = append(out, openaicompat.ContentPart{Type: "text", Text: &txt})
		}
	}
	return out
}

func fromOpenAIResponse(providerName string, r openaicompat.ChatResponse) domain.ChatResponse {
	out := domain.ChatResponse{
		ID:        r.ID,
		Model:     r.Model,
		Provider:  providerName,
		CreatedAt: time.Unix(r.Created, 0).UTC(),
		Usage: domain.Usage{
			PromptTokens:     r.Usage.PromptTokens,
			CompletionTokens: r.Usage.CompletionTokens,
			TotalTokens:      r.Usage.TotalTokens,
		},
	}
	out.Choices = make([]domain.Choice, 0, len(r.Choices))
	for _, c := range r.Choices {
		out.Choices = append(out.Choices, domain.Choice{
			Index:        c.Index,
			Message:      domain.TextMessage(domain.Role(c.Message.Role), c.Message.Content),
			FinishReason: mapFinishReason(c.FinishReason),
		})
	}
	return out
}

func mapFinishReason(s string) domain.FinishReason {
	switch s {
	case "stop":
		return domain.FinishStop
	case "length":
		return domain.FinishLength
	case "content_filter":
		return domain.FinishContentFilter
	case "tool_calls", "function_call":
		return domain.FinishToolCalls
	case "":
		return ""
	default:
		return domain.FinishOther
	}
}
