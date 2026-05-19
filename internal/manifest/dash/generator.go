package dash

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/anupam-chopra/prism/internal/config"
	"github.com/anupam-chopra/prism/internal/media"
	"github.com/anupam-chopra/prism/internal/model"
)

type Generator struct {
	cfg    *config.Config
	logger *slog.Logger
}

func NewGenerator(cfg *config.Config, logger *slog.Logger) *Generator {
	return &Generator{cfg: cfg, logger: logger}
}

type mpdRoot struct {
	XMLName                   xml.Name  `xml:"MPD"`
	XMLNS                     string    `xml:"xmlns,attr"`
	Profiles                  string    `xml:"profiles,attr"`
	Type                      string    `xml:"type,attr"`
	MediaPresentationDuration string    `xml:"mediaPresentationDuration,attr"`
	MinBufferTime             string    `xml:"minBufferTime,attr"`
	Period                    mpdPeriod `xml:"Period"`
}

type mpdPeriod struct {
	AdaptationSets []mpdAdaptationSet `xml:"AdaptationSet"`
}

type mpdAdaptationSet struct {
	ContentType     string              `xml:"contentType,attr"`
	MimeType        string              `xml:"mimeType,attr"`
	Representations []mpdRepresentation `xml:"Representation"`
}

type mpdRepresentation struct {
	ID        string `xml:"id,attr"`
	Bandwidth int    `xml:"bandwidth,attr"`
	Codecs    string `xml:"codecs,attr,omitempty"`
	Width     int    `xml:"width,attr,omitempty"`
	Height    int    `xml:"height,attr,omitempty"`
}

func (g *Generator) Generate(ctx context.Context, req *model.ManifestRequest) ([]byte, error) {
	g.logger.DebugContext(ctx, "generating DASH manifest", "asset_id", req.AssetID)

	if g.cfg.MediaRoot != "" {
		if req.DRM == media.DRMModeClearKey {
			localManifest := media.ClearKeyDASHManifestPath(g.cfg.MediaRoot, req.AssetID)
			if manifestBytes, err := os.ReadFile(localManifest); err == nil {
				return manifestBytes, nil
			} else if !os.IsNotExist(err) {
				return nil, fmt.Errorf("read local ClearKey DASH manifest: %w", err)
			}
		}

		if req.DRM == media.DRMModeWidevine {
			localManifest := media.DRMDASHManifestPath(g.cfg.MediaRoot, req.AssetID)
			if manifestBytes, err := os.ReadFile(localManifest); err == nil {
				return manifestBytes, nil
			} else if !os.IsNotExist(err) {
				return nil, fmt.Errorf("read local DRM DASH manifest: %w", err)
			}
		}

		localManifest := media.DASHManifestPath(g.cfg.MediaRoot, req.AssetID)
		if manifestBytes, err := os.ReadFile(localManifest); err == nil {
			return manifestBytes, nil
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read local DASH manifest: %w", err)
		}
	}

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

	mpd := mpdRoot{
		XMLNS:                     "urn:mpeg:dash:schema:mpd:2011",
		Profiles:                  "urn:mpeg:dash:profile:isoff-live:2011",
		Type:                      "static",
		MediaPresentationDuration: fmt.Sprintf("PT%.1fS", asset.Duration),
		MinBufferTime:             "PT4S",
		Period: mpdPeriod{
			AdaptationSets: []mpdAdaptationSet{
				{
					ContentType:     "video",
					MimeType:        "video/mp4",
					Representations: videoRepresentations(filtered.Renditions),
				},
				{
					ContentType:     "audio",
					MimeType:        "audio/mp4",
					Representations: audioRepresentations(filtered.Tracks),
				},
			},
		},
	}

	xmlBytes, err := xml.MarshalIndent(mpd, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal MPD: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.Write(xmlBytes)

	return Inject(buf.Bytes(), asset), nil
}

func videoRepresentations(renditions []model.Rendition) []mpdRepresentation {
	reps := make([]mpdRepresentation, 0, len(renditions))
	for _, rendition := range renditions {
		width, height := parseResolution(rendition.Resolution)
		reps = append(reps, mpdRepresentation{
			ID:        "video-" + rendition.Resolution,
			Bandwidth: rendition.Bandwidth,
			Codecs:    rendition.Codec,
			Width:     width,
			Height:    height,
		})
	}
	return reps
}

func audioRepresentations(tracks []model.Track) []mpdRepresentation {
	reps := make([]mpdRepresentation, 0, len(tracks))
	for _, track := range tracks {
		if track.Type != "AUDIO" {
			continue
		}
		reps = append(reps, mpdRepresentation{
			ID:        "audio-" + track.Language,
			Bandwidth: 128_000,
			Codecs:    "mp4a.40.2",
		})
	}
	return reps
}

func defaultRenditions() []model.Rendition {
	return []model.Rendition{
		{Bandwidth: 800_000, Resolution: "640x360", Codec: "avc1.42001e", URI: "360p/video.m4s", FrameRate: 24},
		{Bandwidth: 3_000_000, Resolution: "1280x720", Codec: "avc1.4d401f", URI: "720p/video.m4s", FrameRate: 30},
		{Bandwidth: 6_000_000, Resolution: "1920x1080", Codec: "avc1.640028", URI: "1080p/video.m4s", FrameRate: 30},
		{Bandwidth: 16_000_000, Resolution: "3840x2160", Codec: "hvc1.2.4.L153.B0", URI: "2160p/video.m4s", FrameRate: 60},
	}
}

func defaultTracks() []model.Track {
	return []model.Track{
		{Type: "AUDIO", Language: "en", Name: "English", URI: "audio/en/audio.m4s", Default: true},
		{Type: "AUDIO", Language: "es", Name: "Spanish", URI: "audio/es/audio.m4s"},
	}
}

func parseResolution(resolution string) (width, height int) {
	parts := strings.SplitN(resolution, "x", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	_, _ = fmt.Sscanf(parts[0], "%d", &width)
	_, _ = fmt.Sscanf(parts[1], "%d", &height)
	return width, height
}
