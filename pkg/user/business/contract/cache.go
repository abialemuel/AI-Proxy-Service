package contract

import (
	"context"
	"time"
)

type Cache interface {
	Set(ctx context.Context, key string, val interface{}, duration time.Duration) error
	Get(ctx context.Context, key string) (val interface{}, found bool)
	TTL(ctx context.Context, key string) (time.Duration, error)
	Delete(ctx context.Context, key string) error
}
