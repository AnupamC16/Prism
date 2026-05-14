package hls

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/anupam-chopra/prism/internal/config"
	"github.com/anupam-chopra/prism/internal/model"
)

type Generator struct {
	cfg    *config.Config
	logger *slog.Logger
}

func NewGenerator(cfg *config.Config, logger *slog.Logger) *Generator {
	return &Generator{cfg: cfg, logger: logger}
}

func (g *Generator) Generate(ctx context.Context, req *model.ManifestRequest) ([]byte, error) {
	g.logger.DebugContext(ctx, "generating HLS manifest", "asset_id", req.AssetID)

	asset := &model.Asset{
		ID:         req.AssetID,
		Renditions: defaultRenditions(),
		Tracks:     defaultTracks(),
		DRMType:    "widevine",
		KeyID:      "abcd1234ef567890",
		LicenseURL: "https://prism.example.com/license/widevine",
		Duration:   596.4,
	}

	filtered := Filter(asset, req)

	var buf bytes.Buffer
	buf.WriteString("#EXTM3U\n")
	buf.WriteString("#EXT-X-VERSION:6\n")
	buf.WriteString("#EXT-X-INDEPENDENT-SEGMENTS\n\n")

	for _, rendition := range filtered.Renditions {
		fmt.Fprintf(
			&buf,
			"#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%s,CODECS=\"%s\",FRAME-RATE=%.3f\n",
			rendition.Bandwidth,
			rendition.Resolution,
			rendition.Codec,
			rendition.FrameRate,
		)
		fmt.Fprintf(&buf, "%s\n", rendition.URI)
	}

	return Inject(buf.Bytes(), asset), nil
}

func defaultRenditions() []model.Rendition {
	return []model.Rendition{
		{Bandwidth: 800_000, Resolution: "640x360", Codec: "avc1.42001e", URI: "360p/index.m3u8", FrameRate: 24},
		{Bandwidth: 3_000_000, Resolution: "1280x720", Codec: "avc1.4d401f", URI: "720p/index.m3u8", FrameRate: 30},
		{Bandwidth: 6_000_000, Resolution: "1920x1080", Codec: "avc1.640028", URI: "1080p/index.m3u8", FrameRate: 30},
		{Bandwidth: 16_000_000, Resolution: "3840x2160", Codec: "hvc1.2.4.L153.B0", URI: "2160p/index.m3u8", FrameRate: 60},
	}
}

func defaultTracks() []model.Track {
	return []model.Track{
		{Type: "AUDIO", Language: "en", Name: "English", URI: "audio/en/index.m3u8", Default: true},
		{Type: "AUDIO", Language: "es", Name: "Spanish", URI: "audio/es/index.m3u8"},
	}
}
