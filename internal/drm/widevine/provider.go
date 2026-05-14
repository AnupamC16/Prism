package widevine

import (
	"context"
	"log/slog"

	"github.com/anupam-chopra/prism/internal/config"
	"github.com/anupam-chopra/prism/internal/model"
)

type Provider struct {
	client *Client
	logger *slog.Logger
}

func NewProvider(cfg *config.Config, logger *slog.Logger) *Provider {
	return &Provider{
		client: NewClient(cfg.WidevineURL, cfg.WidevineAPIKey, logger),
		logger: logger,
	}
}

func (p *Provider) Name() string { return "widevine" }

func (p *Provider) GetLicense(ctx context.Context, challenge []byte, token string) ([]byte, error) {
	body, statusCode, err := p.client.RequestLicense(ctx, challenge, token)
	if err != nil {
		return nil, model.NewUpstreamError("widevine", statusCode, err.Error())
	}
	return body, nil
}
