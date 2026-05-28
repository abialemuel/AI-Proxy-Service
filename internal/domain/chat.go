// Package domain holds pure, provider-agnostic business types.
// Nothing in this package may import adapters or vendor SDKs.
package domain

import "time"

// Role represents the author of a chat message.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// PartKind discriminates multimodal content parts.
type PartKind string

const (
	PartText     PartKind = "text"
	PartImageURL PartKind = "image_url"
)

// ContentPart is a single piece of multimodal content inside a Message.
type ContentPart struct {
	Kind     PartKind
	Text     string
	ImageURL string // used when Kind == PartImageURL
}

// Message is a provider-agnostic chat message.
type Message struct {
	Role  Role
	Parts []ContentPart
}

// TextMessage is a small constructor helper.
func TextMessage(role Role, text string) Message {
	return Message{Role: role, Parts: []ContentPart{{Kind: PartText, Text: text}}}
}

// ChatRequest is the canonical inbound request to any LLM provider.
type ChatRequest struct {
	Model       string
	Messages    []Message
	Temperature *float64
	TopP        *float64
	MaxTokens   *int
	Stream      bool
	// Provider hint allows overriding the model→provider resolution.
	Provider string
}

// Usage is token accounting normalised across providers.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// FinishReason normalises why a generation stopped.
type FinishReason string

const (
	FinishStop          FinishReason = "stop"
	FinishLength        FinishReason = "length"
	FinishContentFilter FinishReason = "content_filter"
	FinishToolCalls     FinishReason = "tool_calls"
	FinishOther         FinishReason = "other"
)

// Choice is one returned completion alternative.
type Choice struct {
	Index        int
	Message      Message
	FinishReason FinishReason
}

// ChatResponse is the canonical outbound response from any LLM provider.
type ChatResponse struct {
	ID        string
	Model     string
	Provider  string
	CreatedAt time.Time
	Choices   []Choice
	Usage     Usage
}

// StreamChunk is one delta in a streaming response.
type StreamChunk struct {
	Delta        string
	FinishReason FinishReason
	Usage        *Usage // populated on final chunk if known
}
