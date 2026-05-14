package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

const (
	DRMTypeWidevine  = "widevine"
	DRMTypeFairPlay  = "fairplay"
	DRMTypePlayReady = "playready"
)

var ValidDRMTypes = []string{DRMTypeWidevine, DRMTypeFairPlay, DRMTypePlayReady}

type LicenseRequest struct {
	DRMType   string // "widevine", "fairplay", "playready"
	Challenge []byte // raw challenge bytes from EME
	Token     string // DRM JWT token
	AssetID   string // asset being accessed
	SPCBytes  []byte // FairPlay Server Playback Context — FairPlay only
}

func (r *LicenseRequest) Validate() error {
	if !ContainsString(ValidDRMTypes, r.DRMType) {
		return NewValidationError("drm_type", fmt.Sprintf("must be one of %v", ValidDRMTypes))
	}
	if len(r.Challenge) == 0 {
		return NewValidationError("challenge", "is required")
	}
	if r.Token == "" {
		return NewValidationError("token", "is required")
	}
	if r.AssetID == "" {
		return NewValidationError("asset_id", "is required")
	}
	if r.DRMType == DRMTypeFairPlay && len(r.SPCBytes) == 0 {
		return NewValidationError("spc_bytes", "is required for fairplay")
	}
	return nil
}

func (r *LicenseRequest) ChallengeHash() string {
	hash := sha256.Sum256(r.Challenge)
	return hex.EncodeToString(hash[:])
}

type LicenseResponse struct {
	DRMType  string
	License  []byte    // raw license bytes to return to EME
	CachedAt time.Time // when this response was cached
	TTL      int       // seconds until cache expiry
}

func (r *LicenseRequest) ContentType() string {
	switch r.DRMType {
	case DRMTypeWidevine:
		return "application/octet-stream"
	case DRMTypeFairPlay:
		return "application/octet-stream"
	case DRMTypePlayReady:
		return "text/xml; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}
