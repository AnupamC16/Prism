package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anupam-chopra/prism/internal/api/controller"
	"github.com/anupam-chopra/prism/internal/api/response"
	"github.com/anupam-chopra/prism/internal/model"
	"github.com/anupam-chopra/prism/mock"
)

func newTokenController(svc *mock.MockTokenService) *controller.TokenController {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return controller.NewTokenController(svc, log)
}

func decodeEnvelope(t *testing.T, body *bytes.Buffer) response.Envelope {
	t.Helper()

	var env response.Envelope
	if err := json.Unmarshal(body.Bytes(), &env); err != nil {
		t.Fatalf("decode response envelope: %v", err)
	}
	return env
}

func TestIssue_Success(t *testing.T) {
	mockSvc := &mock.MockTokenService{}
	ctrl := newTokenController(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/token", bytes.NewBufferString(`{"asset_id":"test-asset","viewer_id":"test-viewer","ttl":3600}`))
	rec := httptest.NewRecorder()

	ctrl.Issue(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body: %s", rec.Code, rec.Body.String())
	}
	env := decodeEnvelope(t, rec.Body)
	if !env.Success {
		t.Fatalf("expected success envelope, got %+v", env)
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object data, got %T", env.Data)
	}
	if data["token"] != "mock.jwt.token" {
		t.Errorf("expected token mock.jwt.token, got %v", data["token"])
	}
	if data["asset_id"] != "test-asset" {
		t.Errorf("expected asset_id test-asset, got %v", data["asset_id"])
	}
	ttl, ok := data["ttl_seconds"].(float64)
	if !ok || ttl <= 0 {
		t.Errorf("expected ttl_seconds > 0, got %v", data["ttl_seconds"])
	}
}

func TestIssue_InvalidJSON(t *testing.T) {
	mockSvc := &mock.MockTokenService{}
	ctrl := newTokenController(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/token", bytes.NewBufferString(`{"asset_id":}`))
	rec := httptest.NewRecorder()

	ctrl.Issue(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "invalid JSON") {
		t.Errorf("expected invalid JSON message, got: %s", rec.Body.String())
	}
}

func TestIssue_MissingAssetID(t *testing.T) {
	mockSvc := &mock.MockTokenService{}
	ctrl := newTokenController(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/token", bytes.NewBufferString(`{"viewer_id":"test-viewer","ttl":3600}`))
	rec := httptest.NewRecorder()

	ctrl.Issue(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "asset_id") {
		t.Errorf("expected asset_id in body, got: %s", rec.Body.String())
	}
}

func TestIssue_InvalidTTL(t *testing.T) {
	mockSvc := &mock.MockTokenService{}
	ctrl := newTokenController(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/token", bytes.NewBufferString(`{"asset_id":"test-asset","viewer_id":"test-viewer","ttl":30}`))
	rec := httptest.NewRecorder()

	ctrl.Issue(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ttl") {
		t.Errorf("expected ttl in body, got: %s", rec.Body.String())
	}
}

func TestIssue_TTLTooLarge(t *testing.T) {
	mockSvc := &mock.MockTokenService{}
	ctrl := newTokenController(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/token", bytes.NewBufferString(`{"asset_id":"test-asset","viewer_id":"test-viewer","ttl":100000}`))
	rec := httptest.NewRecorder()

	ctrl.Issue(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestIssue_DefaultTTL(t *testing.T) {
	mockSvc := &mock.MockTokenService{}
	ctrl := newTokenController(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/token", bytes.NewBufferString(`{"asset_id":"test-asset","viewer_id":"test-viewer"}`))
	rec := httptest.NewRecorder()

	ctrl.Issue(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body: %s", rec.Code, rec.Body.String())
	}
}

func TestIssue_ServiceError(t *testing.T) {
	mockSvc := &mock.MockTokenService{
		IssueFunc: func(_ context.Context, _ *model.Token) (*model.Token, error) {
			return nil, errors.New("redis down")
		},
	}
	ctrl := newTokenController(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/token", bytes.NewBufferString(`{"asset_id":"test-asset","viewer_id":"test-viewer","ttl":3600}`))
	rec := httptest.NewRecorder()

	ctrl.Issue(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "redis down") {
		t.Errorf("internal error detail must not leak to client: %s", rec.Body.String())
	}
}
