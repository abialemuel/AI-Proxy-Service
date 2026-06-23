// Package ports defines the interfaces (driving and driven) that the
// application core depends on. Adapters implement these.
package ports

import (
	"context"

	"github.com/abialemuel/AI-Proxy-Service/internal/domain"
)

// LLMProvider is the driven port every LLM adapter must implement.
// Adapters translate domain.ChatRequest into their native API and back.
type LLMProvider interface {
	// Name identifies the provider (e.g. "openai", "anthropic", "gemini").
	Name() string

	// Supports reports whether this provider can serve the given model id.
	Supports(model string) bool

	// Chat performs a non-streaming completion.
	Chat(ctx context.Context, req domain.ChatRequest) (domain.ChatResponse, error)

	// Stream performs a streaming completion. Implementations that do not
	// support streaming should return ErrStreamingUnsupported.
	Stream(ctx context.Context, req domain.ChatRequest) (<-chan domain.StreamChunk, error)
}

// ConversationRepository persists per-user conversation context.
type ConversationRepository interface {
	GetByUser(ctx context.Context, userID string) (*domain.Conversation, error)
	Save(ctx context.Context, c *domain.Conversation) error
	Delete(ctx context.Context, userID string) error
}

// Cache is a minimal KV abstraction used for quotas and ephemeral state.
type Cache interface {
	GetInt(ctx context.Context, key string) (int, bool, error)
	IncrBy(ctx context.Context, key string, delta int, ttlSeconds int) (int, error)
	TTL(ctx context.Context, key string) (int, error)
	Del(ctx context.Context, key string) error
}

// OAuthProvider is implemented by SSO providers (Google, Microsoft, ...).
type OAuthProvider interface {
	Name() string
	AuthCodeURL(state string) string
	Exchange(ctx context.Context, code string) (UserInfo, error)
}

// UserInfo is the normalised profile returned by OAuth providers.
type UserInfo struct {
	Email string
	Name  string
	Sub   string
}
