package service

import (
	"context"

	"github.com/anupam-chopra/prism/internal/model"
)

type ManifestServiceI interface {
	GetHLS(ctx context.Context, req *model.ManifestRequest) ([]byte, error)
	GetDASH(ctx context.Context, req *model.ManifestRequest) ([]byte, error)
}

type DRMServiceI interface {
	GetLicense(ctx context.Context, req *model.LicenseRequest) ([]byte, error)
}

type TokenServiceI interface {
	Issue(ctx context.Context, token *model.Token) (*model.Token, error)
	Validate(ctx context.Context, tokenStr string, assetID string) (*model.TokenClaims, error)
	Revoke(ctx context.Context, jti string) error
}
