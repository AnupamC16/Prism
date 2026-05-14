package controller_test

import (
	"context"
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

func TestGetHLS_Success(t *testing.T) {
	mockSvc := &mock.MockManifestService{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctrl := controller.NewManifestController(mockSvc, log)

	req := httptest.NewRequest(http.MethodGet, "/manifest/hls/test-asset-001", nil)
	req.SetPathValue("id", "test-asset-001")
	rec := httptest.NewRecorder()

	ctrl.GetHLS(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "mpegurl") {
		t.Errorf("expected mpegurl content type, got %s", rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), "#EXTM3U") {
		t.Errorf("expected #EXTM3U in body, got: %s", rec.Body.String())
	}
	if len(mockSvc.GetHLSCalls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mockSvc.GetHLSCalls))
	}
	if mockSvc.GetHLSCalls[0].AssetID != "test-asset-001" {
		t.Errorf("expected asset ID test-asset-001, got %s", mockSvc.GetHLSCalls[0].AssetID)
	}
}

func TestGetHLS_MissingAssetID(t *testing.T) {
	mockSvc := &mock.MockManifestService{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctrl := controller.NewManifestController(mockSvc, log)

	req := httptest.NewRequest(http.MethodGet, "/manifest/hls/", nil)
	// no SetPathValue — path value is empty string
	rec := httptest.NewRecorder()

	ctrl.GetHLS(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	if len(mockSvc.GetHLSCalls) != 0 {
		t.Errorf("expected service not called, got %d calls", len(mockSvc.GetHLSCalls))
	}
}

func TestGetHLS_InvalidMaxBandwidth(t *testing.T) {
	mockSvc := &mock.MockManifestService{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctrl := controller.NewManifestController(mockSvc, log)

	req := httptest.NewRequest(http.MethodGet, "/manifest/hls/test-asset?maxBandwidth=notanumber", nil)
	req.SetPathValue("id", "test-asset")
	rec := httptest.NewRecorder()

	ctrl.GetHLS(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "maxBandwidth") {
		t.Errorf("expected 'maxBandwidth' in error body, got: %s", rec.Body.String())
	}
}

func TestGetHLS_NotFound(t *testing.T) {
	mockSvc := &mock.MockManifestService{
		GetHLSFunc: func(_ context.Context, req *model.ManifestRequest) ([]byte, error) {
			return nil, model.NewNotFoundError("manifest", req.AssetID)
		},
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctrl := controller.NewManifestController(mockSvc, log)

	req := httptest.NewRequest(http.MethodGet, "/manifest/hls/missing-asset", nil)
	req.SetPathValue("id", "missing-asset")
	rec := httptest.NewRecorder()

	ctrl.GetHLS(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestGetHLS_InternalError(t *testing.T) {
	mockSvc := &mock.MockManifestService{
		GetHLSFunc: func(_ context.Context, _ *model.ManifestRequest) ([]byte, error) {
			return nil, errors.New("database connection lost")
		},
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctrl := controller.NewManifestController(mockSvc, log)

	req := httptest.NewRequest(http.MethodGet, "/manifest/hls/test-asset", nil)
	req.SetPathValue("id", "test-asset")
	rec := httptest.NewRecorder()

	ctrl.GetHLS(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "database connection lost") {
		t.Errorf("internal error detail must not leak to client: %s", rec.Body.String())
	}
}

func TestGetDASH_Success(t *testing.T) {
	mockSvc := &mock.MockManifestService{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctrl := controller.NewManifestController(mockSvc, log)

	req := httptest.NewRequest(http.MethodGet, "/manifest/dash/test-asset-001", nil)
	req.SetPathValue("id", "test-asset-001")
	rec := httptest.NewRecorder()

	ctrl.GetDASH(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "dash+xml") {
		t.Errorf("expected dash+xml content type, got %s", rec.Header().Get("Content-Type"))
	}
	if len(mockSvc.GetDASHCalls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mockSvc.GetDASHCalls))
	}
	if mockSvc.GetDASHCalls[0].AssetID != "test-asset-001" {
		t.Errorf("expected asset ID test-asset-001, got %s", mockSvc.GetDASHCalls[0].AssetID)
	}
}

func TestGetHLS_WithFilters(t *testing.T) {
	mockSvc := &mock.MockManifestService{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctrl := controller.NewManifestController(mockSvc, log)

	req := httptest.NewRequest(http.MethodGet,
		"/manifest/hls/test-asset?codec=avc1&maxBandwidth=5000000&resolution=1080p", nil)
	req.SetPathValue("id", "test-asset")
	rec := httptest.NewRecorder()

	ctrl.GetHLS(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if len(mockSvc.GetHLSCalls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mockSvc.GetHLSCalls))
	}
	got := mockSvc.GetHLSCalls[0]
	if got.Codec != "avc1" {
		t.Errorf("expected Codec avc1, got %q", got.Codec)
	}
	if got.MaxBandwidth != 5000000 {
		t.Errorf("expected MaxBandwidth 5000000, got %d", got.MaxBandwidth)
	}
	if got.Resolution != "1080p" {
		t.Errorf("expected Resolution 1080p, got %q", got.Resolution)
	}
}
