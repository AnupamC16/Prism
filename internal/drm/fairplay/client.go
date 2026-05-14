package fairplay

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const maxCKCResponseBytes = 4 * 1024 * 1024 // 4MB

type Client struct {
	httpClient *http.Client
	baseURL    string
	secret     string
	logger     *slog.Logger
}

func NewClient(baseURL, secret string, logger *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    baseURL,
		secret:     secret,
		logger:     logger,
	}
}

func (c *Client) RequestCKC(ctx context.Context, spc []byte, drmToken string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(spc))
	if err != nil {
		return nil, 0, fmt.Errorf("build fairplay ckc request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-FairPlay-Secret", c.secret)
	req.Header.Set("X-DRM-Token", drmToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("fairplay ckc request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxCKCResponseBytes))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read fairplay ckc response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, resp.StatusCode, fmt.Errorf("fairplay ksm returned %d: %s", resp.StatusCode, string(body))
	}

	return body, resp.StatusCode, nil
}
