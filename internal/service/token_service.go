package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/anupam-chopra/prism/internal/cache"
	"github.com/anupam-chopra/prism/internal/config"
	"github.com/anupam-chopra/prism/internal/model"
)

type TokenService struct {
	cache     cache.Cache
	jwtSecret []byte
	logger    *slog.Logger
	cfg       *config.Config
}

func NewTokenService(cache cache.Cache, jwtSecret []byte, cfg *config.Config, logger *slog.Logger) *TokenService {
	return &TokenService{
		cache:     cache,
		jwtSecret: jwtSecret,
		logger:    logger,
		cfg:       cfg,
	}
}

func (s *TokenService) Issue(ctx context.Context, token *model.Token) (*model.Token, error) {
	token.JTI = uuid.New().String()
	token.IssuedAt = time.Now().UTC()

	if token.ExpiresAt.IsZero() {
		token.ExpiresAt = time.Now().UTC().Add(time.Duration(s.cfg.TokenTTLSeconds) * time.Second)
	}

	claims := &model.TokenClaims{
		JTI:      token.JTI,
		AssetID:  token.AssetID,
		ViewerID: token.ViewerID,
		Issuer:   model.TokenIssuer,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        token.JTI,
			Subject:   token.ViewerID,
			Issuer:    model.TokenIssuer,
			IssuedAt:  jwt.NewNumericDate(token.IssuedAt),
			ExpiresAt: jwt.NewNumericDate(token.ExpiresAt),
		},
	}

	signedString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	token.SignedString = signedString

	err = s.cache.Set(ctx, cache.TokenKey(token.JTI), []byte(token.AssetID), time.Until(token.ExpiresAt))
	if err != nil {
		s.logger.WarnContext(ctx, "failed to cache token", "jti", token.JTI, "error", err)
	}

	return token, nil
}

func (s *TokenService) Validate(ctx context.Context, tokenStr string, assetID string) (*model.TokenClaims, error) {
	unverifiedClaims := &model.TokenClaims{}
	parser := jwt.NewParser()
	_, _, err := parser.ParseUnverified(tokenStr, unverifiedClaims)
	if err != nil {
		return nil, model.NewTokenError("malformed token")
	}

	jti := unverifiedClaims.JTI
	if jti == "" {
		return nil, model.NewTokenError("malformed token")
	}

	var storedAssetID string
	storedBytes, err := s.cache.Get(ctx, cache.TokenKey(jti))
	if err != nil {
		if errors.Is(err, cache.ErrCacheMiss) {
			return nil, model.NewTokenError("token not found or expired")
		}
		s.logger.WarnContext(ctx, "failed to get token from cache, falling through to JWT verification", "jti", jti, "error", err)
	} else {
		storedAssetID = string(storedBytes)
	}

	tokenClaims := &model.TokenClaims{}
	parsed, err := jwt.ParseWithClaims(tokenStr, tokenClaims, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, model.NewTokenError("unexpected signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil || !parsed.Valid {
		return nil, model.NewTokenError("invalid token")
	}

	if storedAssetID != "" && storedAssetID != assetID {
		return nil, model.NewTokenError("token not valid for this asset")
	}

	if tokenClaims.AssetID != assetID {
		return nil, model.NewTokenError("token not valid for this asset")
	}

	return tokenClaims, nil
}

func (s *TokenService) Revoke(ctx context.Context, jti string) error {
	if jti == "" {
		return model.NewValidationError("jti", "is required")
	}
	return s.cache.Delete(ctx, cache.TokenKey(jti))
}
