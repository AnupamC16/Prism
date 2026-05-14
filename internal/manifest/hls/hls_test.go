package hls

import (
	"context"
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

func TestGenerate_ProducesHLSMasterPlaylistWithWidevineKey(t *testing.T) {
	gen := NewGenerator(&config.Config{}, testLogger())

	got, err := gen.Generate(context.Background(), &model.ManifestRequest{AssetID: "asset-1"})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	body := string(got)

	for _, want := range []string{
		"#EXTM3U",
		"#EXT-X-VERSION:6",
		"#EXT-X-INDEPENDENT-SEGMENTS",
		"#EXT-X-KEY:METHOD=SAMPLE-AES",
		`KEYFORMAT="urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed"`,
		`#EXT-X-STREAM-INF:BANDWIDTH=800000,RESOLUTION=640x360`,
		"360p/index.m3u8",
		"2160p/index.m3u8",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected manifest to contain %q\n%s", want, body)
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

	for _, want := range []string{"360p/index.m3u8", "720p/index.m3u8"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected filtered manifest to contain %q\n%s", want, body)
		}
	}
	for _, notWant := range []string{"1080p/index.m3u8", "2160p/index.m3u8", "hvc1"} {
		if strings.Contains(body, notWant) {
			t.Fatalf("did not expect filtered manifest to contain %q\n%s", notWant, body)
		}
	}
}

func TestFilter_DoesNotMutateOriginalAsset(t *testing.T) {
	asset := &model.Asset{
		ID:         "asset-1",
		Renditions: defaultRenditions(),
		Tracks:     defaultTracks(),
		DRMType:    "fairplay",
		KeyID:      "key-id",
		Duration:   120,
	}
	originalRenditions := len(asset.Renditions)
	originalTracks := len(asset.Tracks)

	filtered := Filter(asset, &model.ManifestRequest{Codec: "hvc1"})
	if len(filtered.Renditions) != 1 {
		t.Fatalf("expected one hvc1 rendition, got %d", len(filtered.Renditions))
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
	base := []byte("#EXTM3U\n#EXT-X-INDEPENDENT-SEGMENTS\n\n")
	tests := []struct {
		name    string
		drmType string
		want    string
	}{
		{name: "none", drmType: "", want: "#EXT-X-INDEPENDENT-SEGMENTS\n\n"},
		{name: "widevine", drmType: "widevine", want: `KEYFORMAT="urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed"`},
		{name: "fairplay", drmType: "fairplay", want: `KEYFORMAT="com.apple.streamingkeydelivery"`},
		{name: "playready", drmType: "playready", want: `KEYFORMAT="com.microsoft.playready"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(Inject(base, &model.Asset{DRMType: tt.drmType, KeyID: "key-id"}))
			if !strings.Contains(got, tt.want) {
				t.Fatalf("expected injected manifest to contain %q, got %s", tt.want, got)
			}
		})
	}
}
