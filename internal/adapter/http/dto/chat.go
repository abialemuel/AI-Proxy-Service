// Package dto holds HTTP request/response shapes and translators to/from
// the application's domain types. DTOs are intentionally separate from
// domain types so wire concerns never leak inward.
package dto

import (
	"github.com/abialemuel/AI-Proxy-Service/internal/domain"
)

// ChatRequest is the public HTTP request body.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages" validate:"required,min=1"`
	Temperature *float64  `json:"temperature,omitempty"`
	TopP        *float64  `json:"top_p,omitempty"`
	MaxTokens   *int      `json:"max_tokens,omitempty"`
	Provider    string    `json:"provider,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// Message can carry either a plain string `content` or a slice of multimodal
// parts. We accept both forms for ergonomic OpenAI parity.
type Message struct {
	Role    string `json:"role" validate:"required,oneof=system user assistant tool"`
	Content any    `json:"content" validate:"required"`
}

// Part is one entry in a multimodal content array.
type Part struct {
	Type     string    `json:"type"`
	Text     *string   `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

// ChatResponse is the public HTTP response body.
type ChatResponse struct {
	ID       string      `json:"id"`
	Provider string      `json:"provider"`
	Model    string      `json:"model"`
	Choices  []ChoiceOut `json:"choices"`
	Usage    UsageOut    `json:"usage"`
}

type ChoiceOut struct {
	Index        int      `json:"index"`
	Message      MsgOut   `json:"message"`
	FinishReason string   `json:"finish_reason,omitempty"`
}

type MsgOut struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type UsageOut struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ToDomain converts a ChatRequest to the canonical domain request.
func (r ChatRequest) ToDomain() domain.ChatRequest {
	out := domain.ChatRequest{
		Model:       r.Model,
		Temperature: r.Temperature,
		TopP:        r.TopP,
		MaxTokens:   r.MaxTokens,
		Stream:      r.Stream,
		Provider:    r.Provider,
	}
	out.Messages = make([]domain.Message, 0, len(r.Messages))
	for _, m := range r.Messages {
		out.Messages = append(out.Messages, toDomainMessage(m))
	}
	return out
}

func toDomainMessage(m Message) domain.Message {
	out := domain.Message{Role: domain.Role(m.Role)}
	switch c := m.Content.(type) {
	case string:
		out.Parts = []domain.ContentPart{{Kind: domain.PartText, Text: c}}
	case []any:
		for _, raw := range c {
			obj, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			t, _ := obj["type"].(string)
			switch t {
			case "image_url":
				if v, ok := obj["image_url"].(map[string]any); ok {
					if url, ok := v["url"].(string); ok {
						out.Parts = append(out.Parts, domain.ContentPart{Kind: domain.PartImageURL, ImageURL: url})
					}
				}
			default:
				if txt, ok := obj["text"].(string); ok {
					out.Parts = append(out.Parts, domain.ContentPart{Kind: domain.PartText, Text: txt})
				}
			}
		}
	default:
		// Fall back to empty user text.
		out.Parts = []domain.ContentPart{{Kind: domain.PartText, Text: ""}}
	}
	return out
}

// FromDomain converts the domain response into the public DTO.
func FromDomain(r domain.ChatResponse) ChatResponse {
	out := ChatResponse{
		ID:       r.ID,
		Provider: r.Provider,
		Model:    r.Model,
		Usage: UsageOut{
			PromptTokens:     r.Usage.PromptTokens,
			CompletionTokens: r.Usage.CompletionTokens,
			TotalTokens:      r.Usage.TotalTokens,
		},
	}
	for _, c := range r.Choices {
		var text string
		for _, p := range c.Message.Parts {
			if p.Kind == domain.PartText {
				text += p.Text
			}
		}
		out.Choices = append(out.Choices, ChoiceOut{
			Index:        c.Index,
			Message:      MsgOut{Role: string(c.Message.Role), Content: text},
			FinishReason: string(c.FinishReason),
		})
	}
	return out
}
