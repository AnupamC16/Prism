package fairplay

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/anupam-chopra/prism/internal/cache"
)

const maxCertResponseBytes = 1 * 1024 * 1024 // 1MB

type CertificateManager struct {
	httpClient *http.Client
	certURL    string
	cache      cache.Cache
	ttl        time.Duration
	logger     *slog.Logger
}

func NewCertificateManager(certURL string, c cache.Cache, ttl time.Duration, logger *slog.Logger) *CertificateManager {
	return &CertificateManager{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		certURL:    certURL,
		cache:      c,
		ttl:        ttl,
		logger:     logger,
	}
}

func (m *CertificateManager) GetCertificate(ctx context.Context) ([]byte, error) {
	key := cache.CertKey("fairplay")

	cached, err := m.cache.Get(ctx, key)
	if err == nil {
		return cached, nil
	}
	if err != cache.ErrCacheMiss {
		m.logger.WarnContext(ctx, "fairplay cert cache get error", "error", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.certURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build fairplay cert request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch fairplay cert failed: %w", err)
	}
	defer resp.Body.Close()

	cert, err := io.ReadAll(io.LimitReader(resp.Body, maxCertResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read fairplay cert response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("fairplay cert server returned %d: %s", resp.StatusCode, string(cert))
	}

	if err := m.cache.Set(ctx, key, cert, m.ttl); err != nil {
		m.logger.WarnContext(ctx, "failed to cache fairplay cert", "error", err)
	}

	return cert, nil
}
