package dash

import (
	"context"
	"encoding/xml"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/anupam-chopra/prism/internal/config"
	"github.com/anupam-chopra/prism/internal/model"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestGenerate_ProducesValidMPDWithWidevineProtection(t *testing.T) {
	gen := NewGenerator(&config.Config{}, testLogger())

	got, err := gen.Generate(context.Background(), &model.ManifestRequest{AssetID: "asset-1"})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	body := string(got)

	var parsed struct {
		XMLName xml.Name `xml:"MPD"`
	}
	if err := xml.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("generated MPD is not valid XML: %v\n%s", err, body)
	}

	for _, want := range []string{
		`profiles="urn:mpeg:dash:profile:isoff-live:2011"`,
		`type="static"`,
		`mediaPresentationDuration="PT596.4S"`,
		`<AdaptationSet contentType="video" mimeType="video/mp4">`,
		`<Representation id="video-1920x1080" bandwidth="6000000" codecs="avc1.640028" width="1920" height="1080"></Representation>`,
		`<Representation id="audio-en" bandwidth="128000" codecs="mp4a.40.2"></Representation>`,
		`schemeIdUri="urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed"`,
		`cenc:default_KID="abcd1234ef567890"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected MPD to contain %q\n%s", want, body)
		}
	}
}

func TestGenerate_AppliesCodecBandwidthAndResolutionFilters(t *testing.T) {
	gen := NewGenerator(&config.Config{}, testLogger())
	req := &model.ManifestRequest{
		AssetID:      "asset-1",
		Codec:        "avc1",
		MaxBandwidth: 3_000_000,
		Resolution:   "720p",
	}

	got, err := gen.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	body := string(got)

	for _, want := range []string{`id="video-640x360"`, `id="video-1280x720"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected filtered MPD to contain %q\n%s", want, body)
		}
	}
	for _, notWant := range []string{`id="video-1920x1080"`, `id="video-3840x2160"`, "hvc1"} {
		if strings.Contains(body, notWant) {
			t.Fatalf("did not expect filtered MPD to contain %q\n%s", notWant, body)
		}
	}
}

func TestFilter_DoesNotMutateOriginalAsset(t *testing.T) {
	asset := &model.Asset{
		ID:         "asset-1",
		Renditions: defaultRenditions(),
		Tracks:     defaultTracks(),
		DRMType:    "widevine",
		KeyID:      "key-id",
		Duration:   120,
	}
	originalRenditions := len(asset.Renditions)
	originalTracks := len(asset.Tracks)

	filtered := Filter(asset, &model.ManifestRequest{Resolution: "720p"})
	if len(filtered.Renditions) != 2 {
		t.Fatalf("expected two renditions at or below 720p, got %d", len(filtered.Renditions))
	}
	filtered.Tracks = append(filtered.Tracks, model.Track{Type: "AUDIO", Language: "fr"})

	if len(asset.Renditions) != originalRenditions {
		t.Fatalf("original renditions mutated: got %d, want %d", len(asset.Renditions), originalRenditions)
	}
	if len(asset.Tracks) != originalTracks {
		t.Fatalf("original tracks mutated: got %d, want %d", len(asset.Tracks), originalTracks)
	}
}

func TestInject_DRMVariants(t *testing.T) {
	base := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011">
  <Period>
    <AdaptationSet contentType="video" mimeType="video/mp4">
    </AdaptationSet>
  </Period>
</MPD>`)
	tests := []struct {
		name    string
		drmType string
		want    string
	}{
		{name: "none", drmType: "", want: `<AdaptationSet contentType="video" mimeType="video/mp4">`},
		{name: "widevine", drmType: "widevine", want: `schemeIdUri="urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed"`},
		{name: "fairplay", drmType: "fairplay", want: `schemeIdUri="urn:uuid:94ce86fb-07ff-4f43-adb8-93d2fa968ca2"`},
		{name: "playready", drmType: "playready", want: `schemeIdUri="urn:uuid:9a04f079-9840-4286-ab92-e65be0885f95"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(Inject(base, &model.Asset{DRMType: tt.drmType, KeyID: "key-id"}))
			if !strings.Contains(got, tt.want) {
				t.Fatalf("expected injected MPD to contain %q, got %s", tt.want, got)
			}
		})
	}
}
