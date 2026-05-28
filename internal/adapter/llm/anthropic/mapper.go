package anthropic

import (
	"strings"
	"time"

	"github.com/abialemuel/AI-Proxy-Service/internal/domain"
)

// toWire converts a domain.ChatRequest to Anthropic's Messages API shape.
// Anthropic separates the system prompt from messages, and only allows
// user / assistant roles in the messages array.
func toWire(r domain.ChatRequest) wireRequest {
	out := wireRequest{
		Model:       r.Model,
		Temperature: r.Temperature,
		TopP:        r.TopP,
		Stream:      r.Stream,
	}
	if r.MaxTokens != nil {
		out.MaxTokens = *r.MaxTokens
	} else {
		out.MaxTokens = 4096 // Anthropic requires max_tokens
	}

	var sysBuilder strings.Builder
	for _, m := range r.Messages {
		if m.Role == domain.RoleSystem {
			if sysBuilder.Len() > 0 {
				sysBuilder.WriteString("\n\n")
			}
			for _, p := range m.Parts {
				if p.Kind == domain.PartText {
					sysBuilder.WriteString(p.Text)
				}
			}
			continue
		}
		wm := wireMessage{Role: string(m.Role)}
		for _, p := range m.Parts {
			switch p.Kind {
			case domain.PartImageURL:
				wm.Content = append(wm.Content, wirePart{
					Type:   "image",
					Source: &wireSource{Type: "url", URL: p.ImageURL},
				})
			default:
				wm.Content = append(wm.Content, wirePart{Type: "text", Text: p.Text})
			}
		}
		out.Messages = append(out.Messages, wm)
	}
	out.System = sysBuilder.String()
	return out
}

func fromWire(providerName string, r wireResponse) domain.ChatResponse {
	var text strings.Builder
	for _, p := range r.Content {
		if p.Type == "text" {
			text.WriteString(p.Text)
		}
	}
	total := r.Usage.InputTokens + r.Usage.OutputTokens
	return domain.ChatResponse{
		ID:        r.ID,
		Model:     r.Model,
		Provider:  providerName,
		CreatedAt: time.Now().UTC(),
		Choices: []domain.Choice{{
			Index:        0,
			Message:      domain.TextMessage(domain.RoleAssistant, text.String()),
			FinishReason: mapStopReason(r.StopReason),
		}},
		Usage: domain.Usage{
			PromptTokens:     r.Usage.InputTokens,
			CompletionTokens: r.Usage.OutputTokens,
			TotalTokens:      total,
		},
	}
}

func mapStopReason(s string) domain.FinishReason {
	switch s {
	case "end_turn", "stop_sequence":
		return domain.FinishStop
	case "max_tokens":
		return domain.FinishLength
	case "tool_use":
		return domain.FinishToolCalls
	case "":
		return ""
	default:
		return domain.FinishOther
	}
}
