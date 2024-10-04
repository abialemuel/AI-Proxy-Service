package redis

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

var rdb *redis.Client

type Redis struct{}

// InitRedis initializes the Redis client
func InitRedis(addr, password string, db int) {
	rdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

// NewGoCache Initialize gocache
func NewRedis() *Redis {
	return &Redis{}
}

// Set sets a key and value in the cache, duration 0 means DefaultExpiration, duration -1 means NoExpiration, duration -2 means KeepTTL
func (gc Redis) Set(ctx context.Context, key string, val interface{}, duration time.Duration) error {
	var err error

	if duration == 0 {
		// No expiration
		err = rdb.Set(ctx, key, val, 0).Err()
	} else if duration == -1 {
		// Keep existing TTL (check version compatibility)
		ttl, err := rdb.TTL(ctx, key).Result()
		if err != nil {
			return err // Handle error retrieving TTL
		}
		if ttl < 0 {
			// No existing key or no TTL set, treat like no expiration
			err = rdb.Set(ctx, key, val, 0).Err()
		} else {
			// Set the new value with the existing TTL
			err = rdb.Set(ctx, key, val, ttl).Err()
		}
	} else {
		// Set with a specified expiration
		err = rdb.Set(ctx, key, val, duration).Err()
	}

	return err
}

// Get retrieves a value from the cache using a key string
func (gc Redis) Get(ctx context.Context, key string) (interface{}, bool) {
	val, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, false
	} else if err != nil {
		return nil, false
	}
	return val, true
}

// retrieves the time to live of a key
func (gc Redis) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl := rdb.TTL(ctx, key).Val()
	return ttl, nil
}

// deletes a key from the cache
func (gc Redis) Delete(ctx context.Context, key string) error {
	err := rdb.Del(ctx, key).Err()
	return err
}
