package gemini

import (
	"strings"
	"time"

	"github.com/abialemuel/AI-Proxy-Service/internal/domain"
)

func toWire(r domain.ChatRequest) wireRequest {
	out := wireRequest{}

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
		role := "user"
		if m.Role == domain.RoleAssistant {
			role = "model"
		}
		c := wireContent{Role: role}
		for _, p := range m.Parts {
			if p.Kind == domain.PartText {
				c.Parts = append(c.Parts, wirePart{Text: p.Text})
			}
		}
		out.Contents = append(out.Contents, c)
	}
	if sysBuilder.Len() > 0 {
		out.SystemInstruction = &wireContent{Parts: []wirePart{{Text: sysBuilder.String()}}}
	}

	if r.Temperature != nil || r.TopP != nil || r.MaxTokens != nil {
		out.GenerationConfig = &wireGenerationConfig{
			Temperature:     r.Temperature,
			TopP:            r.TopP,
			MaxOutputTokens: r.MaxTokens,
		}
	}
	return out
}

func fromWire(providerName, model string, r wireResponse) domain.ChatResponse {
	out := domain.ChatResponse{
		ID:        "",
		Model:     model,
		Provider:  providerName,
		CreatedAt: time.Now().UTC(),
		Usage: domain.Usage{
			PromptTokens:     r.UsageMetadata.PromptTokenCount,
			CompletionTokens: r.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      r.UsageMetadata.TotalTokenCount,
		},
	}
	for _, cand := range r.Candidates {
		var text strings.Builder
		for _, p := range cand.Content.Parts {
			text.WriteString(p.Text)
		}
		out.Choices = append(out.Choices, domain.Choice{
			Index:        cand.Index,
			Message:      domain.TextMessage(domain.RoleAssistant, text.String()),
			FinishReason: mapFinishReason(cand.FinishReason),
		})
	}
	return out
}

func mapFinishReason(s string) domain.FinishReason {
	switch s {
	case "STOP":
		return domain.FinishStop
	case "MAX_TOKENS":
		return domain.FinishLength
	case "SAFETY", "RECITATION":
		return domain.FinishContentFilter
	case "":
		return ""
	default:
		return domain.FinishOther
	}
}
