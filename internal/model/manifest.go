package model

import (
	"fmt"
	"regexp"
)

var (
	ValidCodecs      = []string{"avc1", "hvc1", "av01", "mp4a"}
	ValidResolutions = []string{"360p", "480p", "720p", "1080p", "2160p"}

	assetIDRegex = regexp.MustCompile(`^[a-zA-Z0-9-_]{1,128}$`)
)

type ManifestRequest struct {
	AssetID      string // required — identifies the media asset
	Codec        string // optional — filter by codec: avc1, hvc1, av01, mp4a
	MaxBandwidth int    // optional — filter renditions above this bps
	Resolution   string // optional — max resolution: 360p, 480p, 720p, 1080p, 2160p
	DRM          string // optional — e.g. clearkey for encrypted local DASH
}

func (r *ManifestRequest) FilterHash() string {
	return fmt.Sprintf("%s:%d:%s:%s", r.Codec, r.MaxBandwidth, r.Resolution, r.DRM)
}

func ContainsString(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func IsValidAssetID(assetID string) bool {
	return assetIDRegex.MatchString(assetID)
}

func (r *ManifestRequest) Validate() error {
	if r.AssetID == "" {
		return NewValidationError("asset_id", "is required")
	}
	if !IsValidAssetID(r.AssetID) {
		return NewValidationError("asset_id", "must match regex ^[a-zA-Z0-9-_]{1,128}$")
	}

	if r.Codec != "" && !ContainsString(ValidCodecs, r.Codec) {
		return NewValidationError("codec", fmt.Sprintf("must be one of %v", ValidCodecs))
	}

	if r.Resolution != "" && !ContainsString(ValidResolutions, r.Resolution) {
		return NewValidationError("resolution", fmt.Sprintf("must be one of %v", ValidResolutions))
	}

	if r.MaxBandwidth < 0 {
		return NewValidationError("max_bandwidth", "must be positive")
	}
	if r.DRM != "" && r.DRM != "clearkey" && r.DRM != "widevine" && r.DRM != "fairplay" {
		return NewValidationError("drm", "must be one of clearkey, widevine, fairplay when provided")
	}

	return nil
}

type Rendition struct {
	Bandwidth  int    // bits per second
	Resolution string // e.g. "1920x1080"
	Codec      string // e.g. "avc1.640028"
	URI        string // segment URI — may be CDN-signed
	FrameRate  float64
}

type Track struct {
	Type     string // "AUDIO" or "SUBTITLES"
	Language string // e.g. "en"
	Name     string // e.g. "English"
	URI      string
	Default  bool
}

type Asset struct {
	ID         string
	Renditions []Rendition
	Tracks     []Track
	DRMType    string  // "widevine", "fairplay", "playready", or ""
	KeyID      string  // DRM key ID
	LicenseURL string  // DRM license server URL to inject into manifest
	Duration   float64 // total duration in seconds
}
