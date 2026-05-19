package config

import (
	"strings"
	"testing"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()

	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("JWT_SECRET", "test-secret-32-chars-minimum-value")
	t.Setenv("WIDEVINE_URL", "https://widevine.example.test/license")
	t.Setenv("WIDEVINE_API_KEY", "widevine-key")
	t.Setenv("FAIRPLAY_URL", "https://fairplay.example.test/license")
	t.Setenv("FAIRPLAY_CERT_URL", "https://fairplay.example.test/cert")
	t.Setenv("FAIRPLAY_SECRET", "fairplay-secret")
	t.Setenv("PLAYREADY_URL", "https://playready.example.test/license")
	t.Setenv("CLOUDFRONT_DOMAIN", "")
	t.Setenv("CLOUDFRONT_KEY_PAIR_ID", "")
	t.Setenv("CLOUDFRONT_PRIVATE_KEY", "")
}

func TestLoadAllowsCloudFrontToBeDisabled(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.CloudFrontDomain != "" || cfg.CloudFrontKeyPairID != "" || cfg.CloudFrontPrivateKey != "" {
		t.Fatalf("expected CloudFront config to be empty, got domain=%q key_pair_id=%q private_key=%q",
			cfg.CloudFrontDomain, cfg.CloudFrontKeyPairID, cfg.CloudFrontPrivateKey)
	}
}

func TestLoadRejectsPartialCloudFrontConfig(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("CLOUDFRONT_DOMAIN", "d123456789.cloudfront.net")

	_, err := Load()
	if err == nil {
		t.Fatal("expected partial CloudFront config to fail")
	}
	if !strings.Contains(err.Error(), "incomplete CloudFront configuration") {
		t.Fatalf("expected incomplete CloudFront config error, got %v", err)
	}
}
