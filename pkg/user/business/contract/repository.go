package contract

import (
	"context"

	"github.com/abialemuel/AI-Proxy-Service/pkg/user/modules/repository"
)

type Repository interface {
	// Conversations Repository
	UpsertConversation(ctx context.Context, userID string, message []repository.Message, summary *repository.Summary) error
}
