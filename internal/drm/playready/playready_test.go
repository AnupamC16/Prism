package playready

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anupam-chopra/prism/internal/config"
	"github.com/anupam-chopra/prism/internal/model"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestProviderName(t *testing.T) {
	provider := NewProvider(&config.Config{}, testLogger())

	if got := provider.Name(); got != "playready" {
		t.Fatalf("expected provider name playready, got %q", got)
	}
}

func TestClientRequestLicense_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "text/xml; charset=utf-8" {
			t.Errorf("expected playready XML content type, got %q", got)
		}
		if got := r.Header.Get("SOAPAction"); got != "http://schemas.microsoft.com/DRM/2007/03/protocols/AcquireLicense" {
			t.Errorf("unexpected SOAPAction %q", got)
		}
		if got := r.Header.Get("X-DRM-Token"); got != "test-token" {
			t.Errorf("expected DRM token header, got %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if string(body) != "<challenge/>" {
			t.Errorf("unexpected challenge body %q", body)
		}
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<license/>"))
	}))
	defer server.Close()

	client := NewClient(server.URL, testLogger())
	body, status, err := client.RequestLicense(context.Background(), []byte("<challenge/>"), "test-token")
	if err != nil {
		t.Fatalf("RequestLicense returned error: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if string(body) != "<license/>" {
		t.Fatalf("expected license XML, got %q", body)
	}
}

func TestClientRequestLicense_Non2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewClient(server.URL, testLogger())
	body, status, err := client.RequestLicense(context.Background(), []byte("<challenge/>"), "token")
	if err == nil {
		t.Fatal("expected error")
	}
	if body != nil {
		t.Fatalf("expected nil body on error, got %q", body)
	}
	if status != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", status)
	}
	if !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected status in error, got %v", err)
	}
}

func TestProviderGetLicense_WrapsUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "license server down", http.StatusBadGateway)
	}))
	defer server.Close()

	provider := NewProvider(&config.Config{PlayReadyURL: server.URL}, testLogger())
	_, err := provider.GetLicense(context.Background(), []byte("<challenge/>"), "token")
	if err == nil {
		t.Fatal("expected upstream error")
	}
	if !model.IsUpstreamError(err) {
		t.Fatalf("expected UpstreamError, got %T: %v", err, err)
	}
}
