package hls

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/anupam-chopra/prism/internal/config"
	"github.com/anupam-chopra/prism/internal/media"
	"github.com/anupam-chopra/prism/internal/model"
)

func TestGenerate_ServesLocalFairPlayHLSManifestWhenFairPlayRequested(t *testing.T) {
	root := t.TempDir()
	manifestPath := media.FairPlayHLSManifestPath(root, "asset-fp")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	wantBody := "#EXTM3U\n#EXT-X-KEY:METHOD=SAMPLE-AES,URI=\"skd://fairplay/abc123\",KEYFORMAT=\"com.apple.streamingkeydelivery\"\n"
	if err := os.WriteFile(manifestPath, []byte(wantBody), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	gen := NewGenerator(&config.Config{MediaRoot: root}, testLogger())
	got, err := gen.Generate(context.Background(), &model.ManifestRequest{AssetID: "asset-fp", DRM: media.DRMModeFairPlay})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	if string(got) != wantBody {
		t.Fatalf("expected FairPlay manifest served verbatim (no Inject overwriting EXT-X-KEY)\n got: %s\nwant: %s", got, wantBody)
	}
}
