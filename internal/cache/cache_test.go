package cache_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/anupam-chopra/prism/internal/cache"
)

func setupCache(t *testing.T) (*cache.RedisCache, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rc := cache.NewRedisCache(client, nil)
	t.Cleanup(func() { _ = client.Close() })
	return rc, mr
}

func TestGet_Miss(t *testing.T) {
	rc, _ := setupCache(t)
	_, err := rc.Get(context.Background(), "nonexistent")
	if !errors.Is(err, cache.ErrCacheMiss) {
		t.Fatalf("expected ErrCacheMiss, got %v", err)
	}
}

func TestGet_Hit(t *testing.T) {
	rc, _ := setupCache(t)
	want := []byte("hello prism")
	if err := rc.Set(context.Background(), "key:hit", want, time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := rc.Get(context.Background(), "key:hit")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestSet_And_Get(t *testing.T) {
	rc, _ := setupCache(t)
	want := []byte("manifest-data")
	if err := rc.Set(context.Background(), "key:set", want, 5*time.Second); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := rc.Get(context.Background(), "key:set")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestSet_Expiry(t *testing.T) {
	rc, mr := setupCache(t)
	if err := rc.Set(context.Background(), "key:expiry", []byte("expiring"), time.Second); err != nil {
		t.Fatalf("Set: %v", err)
	}
	mr.FastForward(2 * time.Second)
	_, err := rc.Get(context.Background(), "key:expiry")
	if !errors.Is(err, cache.ErrCacheMiss) {
		t.Fatalf("expected ErrCacheMiss after expiry, got %v", err)
	}
}

func TestDelete_Existing(t *testing.T) {
	rc, _ := setupCache(t)
	if err := rc.Set(context.Background(), "key:del", []byte("value"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := rc.Delete(context.Background(), "key:del"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := rc.Get(context.Background(), "key:del")
	if !errors.Is(err, cache.ErrCacheMiss) {
		t.Fatalf("expected ErrCacheMiss after delete, got %v", err)
	}
}

func TestDelete_NonExistent(t *testing.T) {
	rc, _ := setupCache(t)
	if err := rc.Delete(context.Background(), "key:never"); err != nil {
		t.Fatalf("expected no error deleting non-existent key, got %v", err)
	}
}

func TestExists_True(t *testing.T) {
	rc, _ := setupCache(t)
	if err := rc.Set(context.Background(), "key:exists", []byte("yes"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	ok, err := rc.Exists(context.Background(), "key:exists")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !ok {
		t.Fatal("expected Exists to return true")
	}
}

func TestExists_False(t *testing.T) {
	rc, _ := setupCache(t)
	ok, err := rc.Exists(context.Background(), "key:absent")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if ok {
		t.Fatal("expected Exists to return false")
	}
}

func TestPing_OK(t *testing.T) {
	rc, _ := setupCache(t)
	if err := rc.Ping(context.Background()); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestPing_Unavailable(t *testing.T) {
	rc, mr := setupCache(t)
	mr.Close()
	err := rc.Ping(context.Background())
	if !errors.Is(err, cache.ErrCacheUnavailable) {
		t.Fatalf("expected ErrCacheUnavailable, got %v", err)
	}
}

func TestGet_LargeValue(t *testing.T) {
	rc, _ := setupCache(t)
	want := make([]byte, 1<<20) // 1 MB
	for i := range want {
		want[i] = byte(i % 256)
	}
	if err := rc.Set(context.Background(), "key:large", want, time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := rc.Get(context.Background(), "key:large")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("expected length %d, got %d", len(want), len(got))
	}
}
