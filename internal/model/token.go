package model

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenIssuer = "prism"
	TokenMinTTL = 60    // seconds
	TokenMaxTTL = 86400 // seconds
)

type Token struct {
	JTI          string    // JWT ID — unique identifier for this token
	AssetID      string    // asset this token grants access to
	ViewerID     string    // viewer this token was issued to
	SignedString string    // signed JWT string returned to client
	IssuedAt     time.Time // when token was issued
	ExpiresAt    time.Time // when token expires
}

type TokenClaims struct {
	JTI      string `json:"jti"`
	AssetID  string `json:"asset_id"`
	ViewerID string `json:"viewer_id"`
	Issuer   string `json:"iss"`
	jwt.RegisteredClaims
}

func (t *Token) IsExpired() bool {
	return time.Now().UTC().After(t.ExpiresAt.UTC())
}

func (t *Token) TTLSeconds() int {
	if t.IsExpired() {
		return 0
	}
	return int(t.ExpiresAt.UTC().Sub(time.Now().UTC()).Seconds())
}

func (t *Token) Validate() error {
	if t.AssetID == "" {
		return NewValidationError("asset_id", "is required")
	}
	if t.ViewerID == "" {
		return NewValidationError("viewer_id", "is required")
	}
	if !t.ExpiresAt.After(time.Now()) {
		return NewValidationError("expires_at", "must be in the future")
	}
	return nil
}
