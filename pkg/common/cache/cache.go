package cache

import (
	"context"
	"time"
)

type CacheInterface interface {
	Set(ctx context.Context, key string, val interface{}, duration time.Duration) error
	Get(ctx context.Context, key string) (val interface{}, found bool)
	TTL(ctx context.Context, key string) (time.Duration, error)
	Delete(ctx context.Context, key string) error
}

type Cache struct {
	CacheInterface
}

// NewGoCache Initialize gocache
func NewCache(u CacheInterface) *Cache {
	return &Cache{u}
}
