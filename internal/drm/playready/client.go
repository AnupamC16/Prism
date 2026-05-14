package playready

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
	logger     *slog.Logger
}

func NewClient(baseURL string, logger *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    baseURL,
		logger:     logger,
	}
}

func (c *Client) RequestLicense(ctx context.Context, challenge []byte, drmToken string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(challenge))
	if err != nil {
		return nil, 0, fmt.Errorf("build playready request: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", "http://schemas.microsoft.com/DRM/2007/03/protocols/AcquireLicense")
	req.Header.Set("X-DRM-Token", drmToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("playready request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read playready response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, resp.StatusCode, fmt.Errorf("playready license server returned %d: %s", resp.StatusCode, string(body))
	}

	return body, resp.StatusCode, nil
}
