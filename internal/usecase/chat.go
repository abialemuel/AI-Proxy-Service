// Package usecase contains application services that orchestrate domain logic
// across ports. These are pure and have no transport concerns.
package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/abialemuel/AI-Proxy-Service/internal/domain"
	apierrors "github.com/abialemuel/AI-Proxy-Service/internal/pkg/errors"
	"github.com/abialemuel/AI-Proxy-Service/internal/ports"
)

// ChatService is the high-level chat use case. It:
//   - loads/saves per-user conversation context
//   - enforces a token quota
//   - resolves the right LLM provider via the registry
type ChatService struct {
	registry ports.ProviderRegistry
	repo     ports.ConversationRepository
	quota    *QuotaService
	clock    func() time.Time

	// MaxHistoryMessages caps how many turns of history are sent to providers.
	// When exceeded older messages are dropped (summarisation hook below).
	MaxHistoryMessages int
}

// NewChatService wires a chat service.
func NewChatService(reg ports.ProviderRegistry, repo ports.ConversationRepository, quota *QuotaService) *ChatService {
	return &ChatService{
		registry:           reg,
		repo:               repo,
		quota:              quota,
		clock:              time.Now,
		MaxHistoryMessages: 20,
	}
}

// UserChatInput is the input for a user-scoped chat call (stateful context).
type UserChatInput struct {
	UserID  string
	Request domain.ChatRequest
}

// UserChat runs a chat with persisted context for the given user.
func (s *ChatService) UserChat(ctx context.Context, in UserChatInput) (domain.ChatResponse, error) {
	if strings.TrimSpace(in.UserID) == "" {
		return domain.ChatResponse{}, apierrors.New(apierrors.CodeValidation, "user id required")
	}

	if s.quota != nil {
		if err := s.quota.CheckAndAdmit(ctx, in.UserID); err != nil {
			return domain.ChatResponse{}, err
		}
	}

	conv, err := s.repo.GetByUser(ctx, in.UserID)
	if err != nil {
		return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeUnknown, "load conversation", err)
	}
	if conv == nil {
		conv = &domain.Conversation{
			UserID:    in.UserID,
			CreatedAt: s.clock(),
		}
	}

	conv.Messages = append(conv.Messages, in.Request.Messages...)
	conv.Messages = s.truncate(conv.Messages)

	provider, err := s.registry.Resolve(in.Request)
	if err != nil {
		return domain.ChatResponse{}, err
	}

	outReq := in.Request
	outReq.Messages = conv.Messages
	resp, err := provider.Chat(ctx, outReq)
	if err != nil {
		return domain.ChatResponse{}, err
	}

	// Append assistant response to history and persist.
	if len(resp.Choices) > 0 {
		conv.Messages = append(conv.Messages, resp.Choices[0].Message)
	}
	conv.UpdatedAt = s.clock()
	if err := s.repo.Save(ctx, conv); err != nil {
		// Persistence failure should not fail the call; surface as a wrapped warning.
		return resp, apierrors.Wrap(apierrors.CodeUnknown, "persist conversation", err)
	}

	if s.quota != nil && resp.Usage.TotalTokens > 0 {
		_ = s.quota.RecordUsage(ctx, in.UserID, resp.Usage.TotalTokens)
	}
	return resp, nil
}

// ServiceChat runs a stateless chat for service principals.
func (s *ChatService) ServiceChat(ctx context.Context, req domain.ChatRequest) (domain.ChatResponse, error) {
	provider, err := s.registry.Resolve(req)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	return provider.Chat(ctx, req)
}

// ClearContext drops the persisted conversation for the given user.
func (s *ChatService) ClearContext(ctx context.Context, userID string) error {
	if strings.TrimSpace(userID) == "" {
		return apierrors.New(apierrors.CodeValidation, "user id required")
	}
	return s.repo.Delete(ctx, userID)
}

// truncate keeps the most recent N messages, always preserving the leading
// system messages.
func (s *ChatService) truncate(msgs []domain.Message) []domain.Message {
	if s.MaxHistoryMessages <= 0 || len(msgs) <= s.MaxHistoryMessages {
		return msgs
	}
	var system []domain.Message
	rest := msgs
	for len(rest) > 0 && rest[0].Role == domain.RoleSystem {
		system = append(system, rest[0])
		rest = rest[1:]
	}
	keep := s.MaxHistoryMessages - len(system)
	if keep < 1 {
		keep = 1
	}
	if len(rest) > keep {
		rest = rest[len(rest)-keep:]
	}
	return append(system, rest...)
}

// AvailableProviders reports the registered provider names.
func (s *ChatService) AvailableProviders() []string { return s.registry.List() }

// ensure unused import safety in some build configs
var _ = fmt.Sprintf
