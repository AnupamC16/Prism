package response

import (
	"net/http"
	"time"

	"github.com/anupam-chopra/prism/internal/model"
)

type TokenResponse struct {
	Token     string `json:"token"`
	AssetID   string `json:"asset_id"`
	ExpiresAt string `json:"expires_at"` // RFC3339
	TTL       int    `json:"ttl_seconds"`
}

func NewTokenResponse(t *model.Token) *TokenResponse {
	ttlSeconds := int(time.Until(t.ExpiresAt).Seconds())
	if ttlSeconds < 0 {
		ttlSeconds = 0
	}

	return &TokenResponse{
		Token:     t.SignedString,
		AssetID:   t.AssetID,
		ExpiresAt: t.ExpiresAt.UTC().Format(time.RFC3339),
		TTL:       ttlSeconds,
	}
}

func TokenCreated(w http.ResponseWriter, t *model.Token) {
	Success(w, http.StatusCreated, NewTokenResponse(t))
}
