package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/anupam-chopra/prism/internal/cache"
	"github.com/anupam-chopra/prism/internal/config"
	"github.com/anupam-chopra/prism/internal/drm"
	"github.com/anupam-chopra/prism/internal/model"
)

type DRMService struct {
	router      drm.RouterI
	tokenSvc    TokenServiceI
	cache       cache.Cache
	auditLogger *slog.Logger
	logger      *slog.Logger
	cfg         *config.Config
}

func NewDRMService(router drm.RouterI, tokenSvc TokenServiceI, c cache.Cache, cfg *config.Config, auditLogger *slog.Logger, logger *slog.Logger) *DRMService {
	if router == nil {
		panic(fmt.Sprintf("drm_service: %s is nil", "router"))
	}
	if tokenSvc == nil {
		panic(fmt.Sprintf("drm_service: %s is nil", "tokenSvc"))
	}
	if c == nil {
		panic(fmt.Sprintf("drm_service: %s is nil", "cache"))
	}
	if cfg == nil {
		panic(fmt.Sprintf("drm_service: %s is nil", "cfg"))
	}
	if auditLogger == nil {
		panic(fmt.Sprintf("drm_service: %s is nil", "auditLogger"))
	}
	if logger == nil {
		panic(fmt.Sprintf("drm_service: %s is nil", "logger"))
	}
	return &DRMService{
		router:      router,
		tokenSvc:    tokenSvc,
		cache:       c,
		cfg:         cfg,
		auditLogger: auditLogger,
		logger:      logger,
	}
}

func (s *DRMService) GetLicense(ctx context.Context, req *model.LicenseRequest) ([]byte, error) {
	claims, err := s.tokenSvc.Validate(ctx, req.Token, req.AssetID)
	if err != nil {
		return nil, model.NewTokenError("invalid or expired DRM token")
	}

	cacheKey := cache.LicenseKey(req.DRMType, req.ChallengeHash())

	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		s.auditLogger.Info("license_cache_hit",
			"drm_type", req.DRMType,
			"asset_id", req.AssetID,
			"jti", claims.JTI,
		)
		return cached, nil
	} else if err != cache.ErrCacheMiss {
		s.logger.WarnContext(ctx, "cache get error",
			"drm_type", req.DRMType,
			"asset_id", req.AssetID,
			"error", err,
		)
	}

	provider, err := s.router.Route(req.DRMType)
	if err != nil {
		return nil, model.NewDRMError(req.DRMType, "unsupported DRM type")
	}

	licCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	start := time.Now()
	license, err := provider.GetLicense(licCtx, req.Challenge, req.Token)
	duration := time.Since(start)

	s.auditLogger.Info("license_request",
		"drm_type", req.DRMType,
		"asset_id", req.AssetID,
		"jti", claims.JTI,
		"success", err == nil,
		"duration_ms", duration.Milliseconds(),
		"cache_hit", false,
	)

	if err != nil {
		return nil, model.NewUpstreamError(req.DRMType, 502, err.Error())
	}

	if err := s.cache.Set(ctx, cacheKey, license, time.Duration(s.cfg.LicenseCacheTTL)*time.Second); err != nil {
		s.logger.WarnContext(ctx, "failed to cache license",
			"drm_type", req.DRMType,
			"asset_id", req.AssetID,
			"error", err,
		)
	}

	return license, nil
}
