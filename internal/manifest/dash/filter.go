package dash

import (
	"strings"

	"github.com/anupam-chopra/prism/internal/model"
)

var maxResolutionPixels = map[string]int{
	"360p":  640 * 360,
	"480p":  854 * 480,
	"720p":  1280 * 720,
	"1080p": 1920 * 1080,
	"2160p": 3840 * 2160,
}

var renditionResolutionPixels = map[string]int{
	"640x360":   640 * 360,
	"854x480":   854 * 480,
	"1280x720":  1280 * 720,
	"1920x1080": 1920 * 1080,
	"3840x2160": 3840 * 2160,
}

func Filter(asset *model.Asset, req *model.ManifestRequest) *model.Asset {
	filtered := &model.Asset{
		ID:         asset.ID,
		Renditions: make([]model.Rendition, 0, len(asset.Renditions)),
		Tracks:     append([]model.Track(nil), asset.Tracks...),
		DRMType:    asset.DRMType,
		KeyID:      asset.KeyID,
		LicenseURL: asset.LicenseURL,
		Duration:   asset.Duration,
	}

	maxPixels := maxResolutionPixels[req.Resolution]
	for _, rendition := range asset.Renditions {
		if req.Codec != "" && !strings.HasPrefix(rendition.Codec, req.Codec) {
			continue
		}
		if req.MaxBandwidth != 0 && rendition.Bandwidth > req.MaxBandwidth {
			continue
		}
		if maxPixels != 0 && renditionResolutionPixels[rendition.Resolution] > maxPixels {
			continue
		}
		filtered.Renditions = append(filtered.Renditions, rendition)
	}

	return filtered
}
