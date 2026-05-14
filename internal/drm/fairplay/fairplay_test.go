package fairplay

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/anupam-chopra/prism/internal/cache"
	"github.com/anupam-chopra/prism/internal/config"
	"github.com/anupam-chopra/prism/internal/model"
	"github.com/anupam-chopra/prism/mock"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestProviderName(t *testing.T) {
	provider := NewProvider(&config.Config{}, mock.NewMockCache(), testLogger())

	if got := provider.Name(); got != "fairplay" {
		t.Fatalf("expected provider name fairplay, got %q", got)
	}
}

func TestClientRequestCKC_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/octet-stream" {
			t.Errorf("expected octet-stream content type, got %q", got)
		}
		if got := r.Header.Get("X-FairPlay-Secret"); got != "fairplay-secret" {
			t.Errorf("expected FairPlay secret header, got %q", got)
		}
		if got := r.Header.Get("X-DRM-Token"); got != "test-token" {
			t.Errorf("expected DRM token header, got %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if string(body) != "spc-bytes" {
			t.Errorf("unexpected SPC body %q", body)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ckc-bytes"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "fairplay-secret", testLogger())
	body, status, err := client.RequestCKC(context.Background(), []byte("spc-bytes"), "test-token")
	if err != nil {
		t.Fatalf("RequestCKC returned error: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if string(body) != "ckc-bytes" {
		t.Fatalf("expected CKC bytes, got %q", body)
	}
}

func TestClientRequestCKC_Non2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "ksm down", http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient(server.URL, "fairplay-secret", testLogger())
	body, status, err := client.RequestCKC(context.Background(), []byte("spc"), "token")
	if err == nil {
		t.Fatal("expected error")
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %q", body)
	}
	if status != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", status)
	}
	if !strings.Contains(err.Error(), "502") {
		t.Fatalf("expected status in error, got %v", err)
	}
}

func TestCertificateManagerGetCertificate_CacheHit(t *testing.T) {
	cacheMock := mock.NewMockCache()
	want := []byte("cached-cert")
	if err := cacheMock.Set(context.Background(), cache.CertKey("fairplay"), want, time.Minute); err != nil {
		t.Fatalf("seed cache: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("certificate endpoint should not be called on cache hit")
	}))
	defer server.Close()

	manager := NewCertificateManager(server.URL, cacheMock, time.Minute, testLogger())
	got, err := manager.GetCertificate(context.Background())
	if err != nil {
		t.Fatalf("GetCertificate returned error: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("expected cached cert %q, got %q", want, got)
	}
}

func TestCertificateManagerGetCertificate_CacheMissFetchesAndCaches(t *testing.T) {
	cacheMock := mock.NewMockCache()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fresh-cert"))
	}))
	defer server.Close()

	manager := NewCertificateManager(server.URL, cacheMock, time.Minute, testLogger())
	got, err := manager.GetCertificate(context.Background())
	if err != nil {
		t.Fatalf("GetCertificate returned error: %v", err)
	}
	if string(got) != "fresh-cert" {
		t.Fatalf("expected fresh cert, got %q", got)
	}
	cached, err := cacheMock.Get(context.Background(), cache.CertKey("fairplay"))
	if err != nil {
		t.Fatalf("expected cert cached, got error: %v", err)
	}
	if string(cached) != "fresh-cert" {
		t.Fatalf("expected cached fresh cert, got %q", cached)
	}
}

func TestProviderGetLicense_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cert":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("fairplay-cert"))
		case "/license":
			if got := r.Header.Get("X-FairPlay-Secret"); got != "fairplay-secret" {
				t.Errorf("expected FairPlay secret header, got %q", got)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ckc-bytes"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := NewProvider(&config.Config{
		FairPlayURL:     server.URL + "/license",
		FairPlayCertURL: server.URL + "/cert",
		FairPlaySecret:  "fairplay-secret",
		CertCacheTTL:    60,
	}, mock.NewMockCache(), testLogger())

	body, err := provider.GetLicense(context.Background(), []byte("spc"), "token")
	if err != nil {
		t.Fatalf("GetLicense returned error: %v", err)
	}
	if string(body) != "ckc-bytes" {
		t.Fatalf("expected CKC bytes, got %q", body)
	}
}

func TestProviderGetLicense_WrapsCertificateError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "cert down", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	provider := NewProvider(&config.Config{
		FairPlayURL:     server.URL + "/license",
		FairPlayCertURL: server.URL + "/cert",
		FairPlaySecret:  "fairplay-secret",
		CertCacheTTL:    60,
	}, mock.NewMockCache(), testLogger())

	_, err := provider.GetLicense(context.Background(), []byte("spc"), "token")
	if err == nil {
		t.Fatal("expected upstream error")
	}
	if !model.IsUpstreamError(err) {
		t.Fatalf("expected UpstreamError, got %T: %v", err, err)
	}
}
