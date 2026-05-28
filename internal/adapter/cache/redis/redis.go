// Package redis implements ports.Cache on top of go-redis.
package redis

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/abialemuel/AI-Proxy-Service/internal/ports"
)

// Cache satisfies ports.Cache.
type Cache struct {
	c *redis.Client
}

func New(addr, password string, db int) *Cache {
	return &Cache{c: redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})}
}

// FromClient lets tests inject a redis client.
func FromClient(c *redis.Client) *Cache { return &Cache{c: c} }

func (r *Cache) GetInt(ctx context.Context, key string) (int, bool, error) {
	v, err := r.c.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, false, nil
		}
		return 0, false, err
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false, err
	}
	return n, true, nil
}

func (r *Cache) IncrBy(ctx context.Context, key string, delta int, ttlSeconds int) (int, error) {
	pipe := r.c.Pipeline()
	incr := pipe.IncrBy(ctx, key, int64(delta))
	if ttlSeconds > 0 {
		pipe.Expire(ctx, key, time.Duration(ttlSeconds)*time.Second)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return int(incr.Val()), nil
}

func (r *Cache) TTL(ctx context.Context, key string) (int, error) {
	d, err := r.c.TTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return int(d.Seconds()), nil
}

func (r *Cache) Del(ctx context.Context, key string) error {
	return r.c.Del(ctx, key).Err()
}

// Ping verifies connectivity.
func (r *Cache) Ping(ctx context.Context) error { return r.c.Ping(ctx).Err() }

// Close releases the underlying connection pool.
func (r *Cache) Close() error { return r.c.Close() }

var _ ports.Cache = (*Cache)(nil)
