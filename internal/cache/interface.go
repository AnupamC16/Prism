package cache

import (
	"context"
	"errors"
	"time"
)

var ErrCacheMiss = errors.New("cache: key not found")
var ErrCacheUnavailable = errors.New("cache: service unavailable")

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Ping(ctx context.Context) error
	Close() error
}

type NopCache struct{}

func NewNopCache() *NopCache {
	return &NopCache{}
}

func (n *NopCache) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, ErrCacheMiss
}

func (n *NopCache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return nil
}

func (n *NopCache) Delete(_ context.Context, _ string) error {
	return nil
}

func (n *NopCache) Exists(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (n *NopCache) Ping(_ context.Context) error {
	return nil
}

func (n *NopCache) Close() error {
	return nil
}
