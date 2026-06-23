package usecase

import (
	"context"
	"fmt"

	apierrors "github.com/abialemuel/AI-Proxy-Service/internal/pkg/errors"
	"github.com/abialemuel/AI-Proxy-Service/internal/ports"
)

// QuotaService enforces a rolling-window token budget per user via a Cache.
type QuotaService struct {
	cache      ports.Cache
	windowSec  int
	limit      int
	keyPrefix  string
}

// NewQuotaService wires a quota service. If limit <= 0, the service is a no-op.
func NewQuotaService(cache ports.Cache, windowSec, limit int) *QuotaService {
	return &QuotaService{
		cache:     cache,
		windowSec: windowSec,
		limit:     limit,
		keyPrefix: "quota:user:",
	}
}

func (q *QuotaService) key(userID string) string {
	return q.keyPrefix + userID
}

// CheckAndAdmit returns ErrQuotaExceeded if the user has consumed >= limit
// within the current window. It does not consume tokens itself; call
// RecordUsage after a successful provider call.
func (q *QuotaService) CheckAndAdmit(ctx context.Context, userID string) error {
	if q == nil || q.limit <= 0 || q.cache == nil {
		return nil
	}
	used, _, err := q.cache.GetInt(ctx, q.key(userID))
	if err != nil {
		return nil // fail-open on cache outage
	}
	if used >= q.limit {
		return apierrors.New(apierrors.CodeQuotaExceeded,
			fmt.Sprintf("token quota exceeded (%d/%d) for window %ds", used, q.limit, q.windowSec))
	}
	return nil
}

// RecordUsage adds `tokens` to the user's window counter, initialising the TTL
// on first increment.
func (q *QuotaService) RecordUsage(ctx context.Context, userID string, tokens int) error {
	if q == nil || q.limit <= 0 || q.cache == nil || tokens <= 0 {
		return nil
	}
	_, err := q.cache.IncrBy(ctx, q.key(userID), tokens, q.windowSec)
	return err
}

// Remaining reports the remaining budget for the user.
func (q *QuotaService) Remaining(ctx context.Context, userID string) (int, error) {
	if q == nil || q.limit <= 0 || q.cache == nil {
		return 0, nil
	}
	used, _, err := q.cache.GetInt(ctx, q.key(userID))
	if err != nil {
		return 0, err
	}
	r := q.limit - used
	if r < 0 {
		r = 0
	}
	return r, nil
}
