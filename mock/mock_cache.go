package mock

import (
	"context"
	"sync"
	"time"

	"github.com/anupam-chopra/prism/internal/cache"
)

type MockCache struct {
	data map[string][]byte
	mu   sync.Mutex

	GetFunc    func(ctx context.Context, key string) ([]byte, error)
	SetFunc    func(ctx context.Context, key string, value []byte, ttl time.Duration) error
	DeleteFunc func(ctx context.Context, key string) error
	ExistsFunc func(ctx context.Context, key string) (bool, error)
	PingFunc   func(ctx context.Context) error

	GetCalls    []string
	SetCalls    []struct {
		Key   string
		Value []byte
		TTL   time.Duration
	}
	DeleteCalls []string
}

func NewMockCache() *MockCache {
	return &MockCache{data: make(map[string][]byte)}
}

func (m *MockCache) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	m.GetCalls = append(m.GetCalls, key)
	fn := m.GetFunc
	v, ok := m.data[key]
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, key)
	}
	if !ok {
		return nil, cache.ErrCacheMiss
	}
	return v, nil
}

func (m *MockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	m.SetCalls = append(m.SetCalls, struct {
		Key   string
		Value []byte
		TTL   time.Duration
	}{Key: key, Value: value, TTL: ttl})
	fn := m.SetFunc
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, key, value, ttl)
	}
	m.mu.Lock()
	m.data[key] = value
	m.mu.Unlock()
	return nil
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	m.DeleteCalls = append(m.DeleteCalls, key)
	fn := m.DeleteFunc
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, key)
	}
	m.mu.Lock()
	delete(m.data, key)
	m.mu.Unlock()
	return nil
}

func (m *MockCache) Exists(ctx context.Context, key string) (bool, error) {
	if m.ExistsFunc != nil {
		return m.ExistsFunc(ctx, key)
	}
	m.mu.Lock()
	_, ok := m.data[key]
	m.mu.Unlock()
	return ok, nil
}

func (m *MockCache) Ping(ctx context.Context) error {
	if m.PingFunc != nil {
		return m.PingFunc(ctx)
	}
	return nil
}

func (m *MockCache) Close() error {
	return nil
}

func (m *MockCache) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string][]byte)
	m.GetCalls = nil
	m.SetCalls = nil
	m.DeleteCalls = nil
}
