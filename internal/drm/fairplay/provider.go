package fairplay

import (
	"context"
	"log/slog"
	"time"

	"github.com/anupam-chopra/prism/internal/cache"
	"github.com/anupam-chopra/prism/internal/config"
	"github.com/anupam-chopra/prism/internal/model"
)

type Provider struct {
	client  *Client
	certMgr *CertificateManager
	logger  *slog.Logger
}

func NewProvider(cfg *config.Config, c cache.Cache, logger *slog.Logger) *Provider {
	return &Provider{
		client:  NewClient(cfg.FairPlayURL, cfg.FairPlaySecret, logger),
		certMgr: NewCertificateManager(cfg.FairPlayCertURL, c, time.Duration(cfg.CertCacheTTL)*time.Second, logger),
		logger:  logger,
	}
}

func (p *Provider) Name() string { return "fairplay" }

func (p *Provider) GetLicense(ctx context.Context, challenge []byte, token string) ([]byte, error) {
	if _, err := p.certMgr.GetCertificate(ctx); err != nil {
		return nil, model.NewUpstreamError("fairplay", 0, err.Error())
	}

	body, statusCode, err := p.client.RequestCKC(ctx, challenge, token)
	if err != nil {
		return nil, model.NewUpstreamError("fairplay", statusCode, err.Error())
	}
	return body, nil
}
