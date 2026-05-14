package cache

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	logger *slog.Logger
}

func NewRedisCache(client *redis.Client, logger *slog.Logger) *RedisCache {
	return &RedisCache{client: client, logger: logger}
}

func NewRedisClient(redisURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	return redis.NewClient(opts), nil
}

func (r *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	b, err := r.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, fmt.Errorf("cache get %q: %w", key, err)
	}
	return b, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := r.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("cache set %q: %w", key, err)
	}
	return nil
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("cache delete %q: %w", key, err)
	}
	return nil
}

func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("cache exists %q: %w", key, err)
	}
	return count > 0, nil
}

func (r *RedisCache) Ping(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		return ErrCacheUnavailable
	}
	return nil
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}
