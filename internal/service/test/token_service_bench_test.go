package service_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/anupam-chopra/prism/internal/cache"
	"github.com/anupam-chopra/prism/internal/config"
	"github.com/anupam-chopra/prism/internal/model"
	"github.com/anupam-chopra/prism/internal/service"
)

func setupBenchTokenService(b *testing.B) *service.TokenService {
	b.Helper()

	mr := miniredis.RunT(b)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cache := cache.NewRedisCache(redisClient, logger)
	cfg := &config.Config{TokenTTLSeconds: 3600}
	return service.NewTokenService(cache, []byte("bench-secret-32-chars-minimum!!"), cfg, logger)
}

func BenchmarkTokenIssue(b *testing.B) {
	svc := setupBenchTokenService(b)
	token := &model.Token{AssetID: "bench-asset", ViewerID: "bench-viewer"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.Issue(context.Background(), token)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTokenValidate(b *testing.B) {
	svc := setupBenchTokenService(b)
	token, _ := svc.Issue(context.Background(), &model.Token{
		AssetID:  "bench-asset",
		ViewerID: "bench-viewer",
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.Validate(context.Background(), token.SignedString, "bench-asset")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTokenValidateParallel(b *testing.B) {
	svc := setupBenchTokenService(b)
	token, _ := svc.Issue(context.Background(), &model.Token{
		AssetID:  "bench-asset",
		ViewerID: "bench-viewer",
	})
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := svc.Validate(context.Background(), token.SignedString, "bench-asset")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
