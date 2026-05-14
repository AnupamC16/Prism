package controller_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anupam-chopra/prism/internal/api/controller"
	"github.com/anupam-chopra/prism/internal/model"
	"github.com/anupam-chopra/prism/mock"
)

func newLicenseController(svc *mock.MockDRMService) *controller.LicenseController {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return controller.NewLicenseController(svc, log, log)
}

func TestWidevine_Success(t *testing.T) {
	mockSvc := &mock.MockDRMService{
		GetLicenseFunc: func(_ context.Context, _ *model.LicenseRequest) ([]byte, error) {
			return []byte("mock-license"), nil
		},
	}
	ctrl := newLicenseController(mockSvc)

	body := []byte("test-challenge")
	req := httptest.NewRequest(http.MethodPost, "/license/widevine", bytes.NewReader(body))
	req.Header.Set("X-DRM-Token", "test-token")
	req.Header.Set("X-Asset-ID", "test-asset")
	rec := httptest.NewRecorder()

	ctrl.Widevine(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/octet-stream") {
		t.Errorf("expected octet-stream, got %s", rec.Header().Get("Content-Type"))
	}
	if rec.Body.String() != "mock-license" {
		t.Errorf("expected mock-license, got %q", rec.Body.String())
	}
	if len(mockSvc.GetLicenseCalls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mockSvc.GetLicenseCalls))
	}
	if mockSvc.GetLicenseCalls[0].DRMType != "widevine" {
		t.Errorf("expected DRMType widevine, got %s", mockSvc.GetLicenseCalls[0].DRMType)
	}
}

func TestWidevine_MissingToken(t *testing.T) {
	mockSvc := &mock.MockDRMService{}
	ctrl := newLicenseController(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/license/widevine", bytes.NewReader([]byte("challenge")))
	req.Header.Set("X-Asset-ID", "test-asset")
	// no X-DRM-Token
	rec := httptest.NewRecorder()

	ctrl.Widevine(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "X-DRM-Token") {
		t.Errorf("expected X-DRM-Token in body, got: %s", rec.Body.String())
	}
}

func TestWidevine_MissingAssetID(t *testing.T) {
	mockSvc := &mock.MockDRMService{}
	ctrl := newLicenseController(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/license/widevine", bytes.NewReader([]byte("challenge")))
	req.Header.Set("X-DRM-Token", "test-token")
	// no X-Asset-ID
	rec := httptest.NewRecorder()

	ctrl.Widevine(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestWidevine_EmptyBody(t *testing.T) {
	mockSvc := &mock.MockDRMService{}
	ctrl := newLicenseController(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/license/widevine", bytes.NewReader(nil))
	req.Header.Set("X-DRM-Token", "test-token")
	req.Header.Set("X-Asset-ID", "test-asset")
	rec := httptest.NewRecorder()

	ctrl.Widevine(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "challenge") {
		t.Errorf("expected 'challenge' in body, got: %s", rec.Body.String())
	}
}

func TestWidevine_TokenError(t *testing.T) {
	mockSvc := &mock.MockDRMService{
		GetLicenseFunc: func(_ context.Context, _ *model.LicenseRequest) ([]byte, error) {
			return nil, model.NewTokenError("expired token")
		},
	}
	ctrl := newLicenseController(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/license/widevine", bytes.NewReader([]byte("challenge")))
	req.Header.Set("X-DRM-Token", "test-token")
	req.Header.Set("X-Asset-ID", "test-asset")
	rec := httptest.NewRecorder()

	ctrl.Widevine(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestWidevine_UpstreamError(t *testing.T) {
	mockSvc := &mock.MockDRMService{
		GetLicenseFunc: func(_ context.Context, _ *model.LicenseRequest) ([]byte, error) {
			return nil, model.NewUpstreamError("widevine", 503, "service down")
		},
	}
	ctrl := newLicenseController(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/license/widevine", bytes.NewReader([]byte("challenge")))
	req.Header.Set("X-DRM-Token", "test-token")
	req.Header.Set("X-Asset-ID", "test-asset")
	rec := httptest.NewRecorder()

	ctrl.Widevine(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "widevine") {
		t.Errorf("expected 'widevine' in body, got: %s", rec.Body.String())
	}
}

func TestWidevine_BodyTooLarge(t *testing.T) {
	mockSvc := &mock.MockDRMService{}
	ctrl := newLicenseController(mockSvc)

	body := bytes.Repeat([]byte("x"), 2*1024*1024+1)
	req := httptest.NewRequest(http.MethodPost, "/license/widevine", bytes.NewReader(body))
	req.Header.Set("X-DRM-Token", "test-token")
	req.Header.Set("X-Asset-ID", "test-asset")
	rec := httptest.NewRecorder()

	ctrl.Widevine(rec, req)

	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 400 or 413 for oversized body, got %d", rec.Code)
	}
}

func TestFairPlay_Success(t *testing.T) {
	spc := []byte("spc-bytes")
	spcBase64 := base64.StdEncoding.EncodeToString(spc)

	var gotSPC []byte
	mockSvc := &mock.MockDRMService{
		GetLicenseFunc: func(_ context.Context, req *model.LicenseRequest) ([]byte, error) {
			gotSPC = req.SPCBytes
			return []byte("mock-ckc"), nil
		},
	}
	ctrl := newLicenseController(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/license/fairplay", bytes.NewReader([]byte("challenge")))
	req.Header.Set("X-DRM-Token", "test-token")
	req.Header.Set("X-Asset-ID", "test-asset")
	req.Header.Set("X-FairPlay-SPC", spcBase64)
	rec := httptest.NewRecorder()

	ctrl.FairPlay(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if !bytes.Equal(gotSPC, spc) {
		t.Errorf("SPC bytes mismatch: got %q, want %q", gotSPC, spc)
	}
}

func TestFairPlay_InvalidSPC(t *testing.T) {
	mockSvc := &mock.MockDRMService{}
	ctrl := newLicenseController(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/license/fairplay", bytes.NewReader([]byte("challenge")))
	req.Header.Set("X-DRM-Token", "test-token")
	req.Header.Set("X-Asset-ID", "test-asset")
	req.Header.Set("X-FairPlay-SPC", "not-valid-base64!@#")
	rec := httptest.NewRecorder()

	ctrl.FairPlay(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "X-FairPlay-SPC") {
		t.Errorf("expected 'X-FairPlay-SPC' in body, got: %s", rec.Body.String())
	}
}

func TestPlayReady_Success(t *testing.T) {
	mockSvc := &mock.MockDRMService{}
	ctrl := newLicenseController(mockSvc)

	xmlBody := []byte(`<?xml version="1.0"?><licenseRequest/>`)
	req := httptest.NewRequest(http.MethodPost, "/license/playready", bytes.NewReader(xmlBody))
	req.Header.Set("X-DRM-Token", "test-token")
	req.Header.Set("X-Asset-ID", "test-asset")
	rec := httptest.NewRecorder()

	ctrl.PlayReady(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/xml") {
		t.Errorf("expected text/xml content type, got %s", rec.Header().Get("Content-Type"))
	}
	if len(mockSvc.GetLicenseCalls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mockSvc.GetLicenseCalls))
	}
	if mockSvc.GetLicenseCalls[0].DRMType != "playready" {
		t.Errorf("expected DRMType playready, got %s", mockSvc.GetLicenseCalls[0].DRMType)
	}
}

func TestLicenseController_UnknownError(t *testing.T) {
	mockSvc := &mock.MockDRMService{
		GetLicenseFunc: func(_ context.Context, _ *model.LicenseRequest) ([]byte, error) {
			return nil, errors.New("unexpected internal failure")
		},
	}
	ctrl := newLicenseController(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/license/widevine", bytes.NewReader([]byte("challenge")))
	req.Header.Set("X-DRM-Token", "test-token")
	req.Header.Set("X-Asset-ID", "test-asset")
	rec := httptest.NewRecorder()

	ctrl.Widevine(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "unexpected internal failure") {
		t.Errorf("internal error detail must not leak to client")
	}
}
