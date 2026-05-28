package domain

import "time"

// Conversation stores rolling chat context per user.
type Conversation struct {
	ID        string
	UserID    string
	Messages  []Message
	Summary   string // running summary used to compress old turns
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TokenQuota tracks per-user consumption inside a rolling window.
type TokenQuota struct {
	UserID    string
	Used      int
	Limit     int
	WindowSec int
	ResetAt   time.Time
}

// Remaining returns the tokens still available; never negative.
func (q TokenQuota) Remaining() int {
	if q.Limit <= 0 {
		return 0
	}
	r := q.Limit - q.Used
	if r < 0 {
		return 0
	}
	return r
}
