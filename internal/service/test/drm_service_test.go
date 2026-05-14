package service_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/anupam-chopra/prism/internal/cache"
	"github.com/anupam-chopra/prism/internal/config"
	"github.com/anupam-chopra/prism/internal/drm"
	"github.com/anupam-chopra/prism/internal/model"
	"github.com/anupam-chopra/prism/internal/service"
	"github.com/anupam-chopra/prism/mock"
)

func setupDRMService(t *testing.T) (*service.DRMService, *mock.MockCache, *mock.MockTokenService, *mock.MockDRMProvider, *bytes.Buffer) {
	t.Helper()

	auditBuf := &bytes.Buffer{}
	auditLogger := slog.New(slog.NewTextHandler(auditBuf, nil))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cacheMock := mock.NewMockCache()
	tokenSvc := &mock.MockTokenService{}
	provider := &mock.MockDRMProvider{NameValue: "widevine"}
	router := drm.NewRouter(
		provider,
		&mock.MockDRMProvider{NameValue: "fairplay"},
		&mock.MockDRMProvider{NameValue: "playready"},
	)
	cfg := &config.Config{LicenseCacheTTL: 300}
	svc := service.NewDRMService(router, tokenSvc, cacheMock, cfg, auditLogger, logger)
	return svc, cacheMock, tokenSvc, provider, auditBuf
}

func testLicenseRequest() *model.LicenseRequest {
	return &model.LicenseRequest{
		DRMType:   model.DRMTypeWidevine,
		Challenge: []byte("challenge"),
		Token:     "token",
		AssetID:   "asset-1",
	}
}

func TestGetLicense_Success(t *testing.T) {
	svc, _, _, provider, auditBuf := setupDRMService(t)
	req := testLicenseRequest()

	got, err := svc.GetLicense(context.Background(), req)
	if err != nil {
		t.Fatalf("GetLicense returned error: %v", err)
	}
	if !bytes.Equal(got, []byte("mock-license")) {
		t.Fatalf("expected mock-license, got %q", got)
	}
	if len(provider.GetLicenseCalls) != 1 {
		t.Fatalf("expected provider called once, got %d calls", len(provider.GetLicenseCalls))
	}
	if auditBuf.Len() == 0 {
		t.Fatal("expected audit log to be written")
	}
}

func TestGetLicense_InvalidToken(t *testing.T) {
	svc, _, tokenSvc, provider, _ := setupDRMService(t)
	tokenSvc.ValidateFunc = func(_ context.Context, _, _ string) (*model.TokenClaims, error) {
		return nil, model.NewTokenError("expired")
	}

	_, err := svc.GetLicense(context.Background(), testLicenseRequest())
	if err == nil || !model.IsTokenError(err) {
		t.Fatalf("expected TokenError, got %v", err)
	}
	if len(provider.GetLicenseCalls) != 0 {
		t.Fatalf("expected provider not called, got %d calls", len(provider.GetLicenseCalls))
	}
}

func TestGetLicense_UnsupportedDRMType(t *testing.T) {
	svc, _, _, provider, _ := setupDRMService(t)
	req := testLicenseRequest()
	req.DRMType = "unknown"

	_, err := svc.GetLicense(context.Background(), req)
	if err == nil || !model.IsDRMError(err) {
		t.Fatalf("expected DRMError, got %v", err)
	}
	if len(provider.GetLicenseCalls) != 0 {
		t.Fatalf("expected provider not called, got %d calls", len(provider.GetLicenseCalls))
	}
}

func TestGetLicense_CacheHit(t *testing.T) {
	svc, cacheMock, _, provider, _ := setupDRMService(t)
	req := testLicenseRequest()
	want := []byte("cached-license")
	key := cache.LicenseKey(req.DRMType, req.ChallengeHash())
	if err := cacheMock.Set(context.Background(), key, want, time.Minute); err != nil {
		t.Fatalf("seed cache: %v", err)
	}

	got, err := svc.GetLicense(context.Background(), req)
	if err != nil {
		t.Fatalf("GetLicense returned error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("expected cached bytes %q, got %q", want, got)
	}
	if len(provider.GetLicenseCalls) != 0 {
		t.Fatalf("expected provider not called, got %d calls", len(provider.GetLicenseCalls))
	}
}

func TestGetLicense_UpstreamError(t *testing.T) {
	svc, _, _, provider, _ := setupDRMService(t)
	provider.GetLicenseFunc = func(_ context.Context, _ []byte, _ string) ([]byte, error) {
		return nil, errors.New("provider down")
	}

	_, err := svc.GetLicense(context.Background(), testLicenseRequest())
	if err == nil || !model.IsUpstreamError(err) {
		t.Fatalf("expected UpstreamError, got %v", err)
	}
}

func TestGetLicense_CachesResponse(t *testing.T) {
	svc, cacheMock, _, _, _ := setupDRMService(t)
	req := testLicenseRequest()
	key := cache.LicenseKey(req.DRMType, req.ChallengeHash())

	_, err := svc.GetLicense(context.Background(), req)
	if err != nil {
		t.Fatalf("GetLicense returned error: %v", err)
	}
	for _, call := range cacheMock.SetCalls {
		if call.Key == key && bytes.Equal(call.Value, []byte("mock-license")) {
			return
		}
	}
	t.Fatalf("expected cache set for key %q, got %+v", key, cacheMock.SetCalls)
}
