package cdn

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/anupam-chopra/prism/internal/config"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestDisabledCloudFrontLeavesURLsAndManifestsUnchanged(t *testing.T) {
	client, err := NewCloudFront(&config.Config{}, discardLogger())
	if err != nil {
		t.Fatalf("NewCloudFront returned error: %v", err)
	}

	const originalURL = "https://origin.example.test/assets/test/video.m4s"
	signedURL, err := client.SignURL(originalURL, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("SignURL returned error: %v", err)
	}
	if signedURL != originalURL {
		t.Fatalf("expected URL to be unchanged, got %q", signedURL)
	}

	manifest := []byte("#EXTM3U\nsegment-001.ts\n")
	rewritten, err := client.RewriteManifestURIs(context.Background(), manifest, "asset-1")
	if err != nil {
		t.Fatalf("RewriteManifestURIs returned error: %v", err)
	}
	if string(rewritten) != string(manifest) {
		t.Fatalf("expected manifest to be unchanged, got %q", rewritten)
	}
}
