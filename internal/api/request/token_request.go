package request

import (
	"regexp"
	"time"

	"github.com/anupam-chopra/prism/internal/model"
)

var tokenAssetIDRegex = regexp.MustCompile(`^[a-zA-Z0-9-_]{1,128}$`)

type TokenRequest struct {
	AssetID  string `json:"asset_id"`
	ViewerID string `json:"viewer_id"`
	TTL      int    `json:"ttl"`
}

type RevokeTokenRequest struct {
	JTI string `json:"jti"`
}

func (r *TokenRequest) Validate() error {
	if r.AssetID == "" {
		return model.NewValidationError("asset_id", "is required")
	}
	if len(r.AssetID) > 128 {
		return model.NewValidationError("asset_id", "must be at most 128 characters")
	}
	if !tokenAssetIDRegex.MatchString(r.AssetID) {
		return model.NewValidationError("asset_id", "must match ^[a-zA-Z0-9-_]{1,128}$")
	}
	if r.ViewerID == "" {
		return model.NewValidationError("viewer_id", "is required")
	}
	if len(r.ViewerID) > 128 {
		return model.NewValidationError("viewer_id", "must be at most 128 characters")
	}
	if r.TTL != 0 && (r.TTL < 60 || r.TTL > 86400) {
		return model.NewValidationError("ttl", "must be between 60 and 86400 seconds")
	}
	return nil
}

func (r *TokenRequest) ToDomain() *model.Token {
	token := &model.Token{
		AssetID:  r.AssetID,
		ViewerID: r.ViewerID,
	}
	if r.TTL > 0 {
		token.ExpiresAt = time.Now().UTC().Add(time.Duration(r.TTL) * time.Second)
	}
	return token
}

func (r *RevokeTokenRequest) Validate() error {
	if r.JTI == "" {
		return model.NewValidationError("jti", "is required")
	}
	return nil
}
