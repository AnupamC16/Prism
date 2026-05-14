package widevine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const maxResponseBytes = 4 * 1024 * 1024 // 4MB

type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	logger     *slog.Logger
}

func NewClient(baseURL, apiKey string, logger *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    baseURL,
		apiKey:     apiKey,
		logger:     logger,
	}
}

func (c *Client) RequestLicense(ctx context.Context, challenge []byte, drmToken string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(challenge))
	if err != nil {
		return nil, 0, fmt.Errorf("build widevine request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("X-DRM-Token", drmToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("widevine request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read widevine response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, resp.StatusCode, fmt.Errorf("widevine license server returned %d: %s", resp.StatusCode, string(body))
	}

	return body, resp.StatusCode, nil
}
