package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/anupam-chopra/prism/internal/cache"
	"github.com/anupam-chopra/prism/internal/config"
	"github.com/anupam-chopra/prism/internal/model"
	"github.com/anupam-chopra/prism/internal/service"
)

func setupTokenService(t *testing.T) (*service.TokenService, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = redisClient.Close()
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	redisCache := cache.NewRedisCache(redisClient, logger)
	cfg := &config.Config{TokenTTLSeconds: 3600}
	svc := service.NewTokenService(redisCache, []byte("test-secret-32-chars-minimum!!"), cfg, logger)
	return svc, mr
}

func TestIssue_Success(t *testing.T) {
	svc, mr := setupTokenService(t)

	token, err := svc.Issue(context.Background(), &model.Token{
		AssetID:  "asset-1",
		ViewerID: "viewer-1",
	})
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}
	if token.JTI == "" {
		t.Fatal("expected non-empty JTI")
	}
	if token.SignedString == "" {
		t.Fatal("expected non-empty signed token")
	}
	if !token.ExpiresAt.After(time.Now().UTC()) {
		t.Fatalf("expected future ExpiresAt, got %s", token.ExpiresAt)
	}
	if !mr.Exists(cache.TokenKey(token.JTI)) {
		t.Fatalf("expected Redis key %q to exist", cache.TokenKey(token.JTI))
	}
}

func TestValidate_Success(t *testing.T) {
	svc, _ := setupTokenService(t)
	token, err := svc.Issue(context.Background(), &model.Token{
		AssetID:  "asset-1",
		ViewerID: "viewer-1",
	})
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}

	claims, err := svc.Validate(context.Background(), token.SignedString, "asset-1")
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if claims.JTI != token.JTI {
		t.Errorf("expected JTI %q, got %q", token.JTI, claims.JTI)
	}
	if claims.AssetID != "asset-1" {
		t.Errorf("expected AssetID asset-1, got %q", claims.AssetID)
	}
	if claims.ViewerID != "viewer-1" {
		t.Errorf("expected ViewerID viewer-1, got %q", claims.ViewerID)
	}
}

func TestValidate_WrongAssetID(t *testing.T) {
	svc, _ := setupTokenService(t)
	token, err := svc.Issue(context.Background(), &model.Token{
		AssetID:  "asset-A",
		ViewerID: "viewer-1",
	})
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}

	_, err = svc.Validate(context.Background(), token.SignedString, "asset-B")
	if err == nil || !model.IsTokenError(err) {
		t.Fatalf("expected TokenError, got %v", err)
	}
}

func TestValidate_ExpiredViaRedis(t *testing.T) {
	svc, mr := setupTokenService(t)
	token, err := svc.Issue(context.Background(), &model.Token{
		AssetID:   "asset-1",
		ViewerID:  "viewer-1",
		ExpiresAt: time.Now().UTC().Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}

	mr.FastForward(3 * time.Second)
	_, err = svc.Validate(context.Background(), token.SignedString, "asset-1")
	if err == nil || !model.IsTokenError(err) {
		t.Fatalf("expected TokenError, got %v", err)
	}
	if !strings.Contains(err.Error(), "token not found or expired") {
		t.Fatalf("expected token not found or expired, got %v", err)
	}
}

func TestValidate_MalformedToken(t *testing.T) {
	svc, _ := setupTokenService(t)

	_, err := svc.Validate(context.Background(), "not.a.valid.jwt", "asset-1")
	if err == nil || !model.IsTokenError(err) {
		t.Fatalf("expected TokenError, got %v", err)
	}
}

func TestValidate_TamperedToken(t *testing.T) {
	svc, _ := setupTokenService(t)
	token, err := svc.Issue(context.Background(), &model.Token{
		AssetID:  "asset-1",
		ViewerID: "viewer-1",
	})
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}
	tampered := token.SignedString[:len(token.SignedString)-5] + "xxxxx"

	_, err = svc.Validate(context.Background(), tampered, "asset-1")
	if err == nil || !model.IsTokenError(err) {
		t.Fatalf("expected TokenError, got %v", err)
	}
	if !strings.Contains(err.Error(), "invalid token") {
		t.Fatalf("expected invalid token, got %v", err)
	}
}

func TestRevoke_Success(t *testing.T) {
	svc, _ := setupTokenService(t)
	token, err := svc.Issue(context.Background(), &model.Token{
		AssetID:  "asset-1",
		ViewerID: "viewer-1",
	})
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}

	if err := svc.Revoke(context.Background(), token.JTI); err != nil {
		t.Fatalf("Revoke returned error: %v", err)
	}
	_, err = svc.Validate(context.Background(), token.SignedString, "asset-1")
	if err == nil || !model.IsTokenError(err) {
		t.Fatalf("expected TokenError after revoke, got %v", err)
	}
}

func TestRevoke_NonexistentJTI(t *testing.T) {
	svc, _ := setupTokenService(t)

	if err := svc.Revoke(context.Background(), "unknown-jti"); err != nil {
		t.Fatalf("expected no error for nonexistent JTI, got %v", err)
	}
}

func TestRevoke_EmptyJTI(t *testing.T) {
	svc, _ := setupTokenService(t)

	err := svc.Revoke(context.Background(), "")
	if err == nil {
		t.Fatal("expected validation error")
	}
	var ve *model.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}
