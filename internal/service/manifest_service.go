package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/anupam-chopra/prism/internal/cache"
	"github.com/anupam-chopra/prism/internal/cdn"
	"github.com/anupam-chopra/prism/internal/config"
	"github.com/anupam-chopra/prism/internal/manifest"
	"github.com/anupam-chopra/prism/internal/model"
)

type ManifestService struct {
	hlsGen  manifest.GeneratorI
	dashGen manifest.GeneratorI
	cache   cache.Cache
	cdn     *cdn.CloudFront
	logger  *slog.Logger
	cfg     *config.Config
}

func NewManifestService(hlsGen manifest.GeneratorI, dashGen manifest.GeneratorI, c cache.Cache, cdnClient *cdn.CloudFront, cfg *config.Config, logger *slog.Logger) *ManifestService {
	if hlsGen == nil {
		panic("manifest_service: hlsGen is nil")
	}
	if dashGen == nil {
		panic("manifest_service: dashGen is nil")
	}
	if c == nil {
		panic("manifest_service: cache is nil")
	}
	if cdnClient == nil {
		panic("manifest_service: cdn is nil")
	}
	if cfg == nil {
		panic("manifest_service: cfg is nil")
	}
	if logger == nil {
		panic("manifest_service: logger is nil")
	}
	return &ManifestService{
		hlsGen:  hlsGen,
		dashGen: dashGen,
		cache:   c,
		cdn:     cdnClient,
		logger:  logger,
		cfg:     cfg,
	}
}

func (s *ManifestService) GetHLS(ctx context.Context, req *model.ManifestRequest) ([]byte, error) {
	return s.get(ctx, "hls", req, s.hlsGen)
}

func (s *ManifestService) GetDASH(ctx context.Context, req *model.ManifestRequest) ([]byte, error) {
	return s.get(ctx, "dash", req, s.dashGen)
}

func (s *ManifestService) get(ctx context.Context, manifestType string, req *model.ManifestRequest, gen manifest.GeneratorI) ([]byte, error) {
	key := cache.ManifestKey(manifestType, req.AssetID, req.FilterHash())

	if cached, err := s.cache.Get(ctx, key); err == nil {
		s.logger.DebugContext(ctx, "manifest cache hit",
			"asset_id", req.AssetID,
			"type", manifestType,
		)
		return cached, nil
	} else if err != cache.ErrCacheMiss {
		s.logger.WarnContext(ctx, "cache get error",
			"asset_id", req.AssetID,
			"type", manifestType,
			"error", err,
		)
	}

	manifestBytes, err := gen.Generate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("generate %s manifest for asset %q: %w", manifestType, req.AssetID, err)
	}

	manifestBytes, err = s.cdn.RewriteManifestURIs(ctx, manifestBytes, req.AssetID)
	if err != nil {
		return nil, fmt.Errorf("rewrite CDN URIs for asset %q: %w", req.AssetID, err)
	}

	if err := s.cache.Set(ctx, key, manifestBytes, time.Duration(s.cfg.ManifestCacheTTL)*time.Second); err != nil {
		s.logger.WarnContext(ctx, "failed to cache manifest",
			"asset_id", req.AssetID,
			"type", manifestType,
			"error", err,
		)
	}

	s.logger.InfoContext(ctx, "manifest cache miss — generated",
		"asset_id", req.AssetID,
		"type", manifestType,
		"size_bytes", len(manifestBytes),
	)

	return manifestBytes, nil
}
