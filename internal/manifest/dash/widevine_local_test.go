package dash

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/anupam-chopra/prism/internal/config"
	"github.com/anupam-chopra/prism/internal/media"
	"github.com/anupam-chopra/prism/internal/model"
)

func TestGenerate_ServesLocalDRMDASHManifestWhenWidevineRequested(t *testing.T) {
	root := t.TempDir()
	manifestPath := media.DRMDASHManifestPath(root, "asset-wv")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	wantBody := `<?xml version="1.0"?><MPD><ContentProtection schemeIdUri="urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed"><cenc:pssh>AAAA</cenc:pssh></ContentProtection></MPD>`
	if err := os.WriteFile(manifestPath, []byte(wantBody), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	gen := NewGenerator(&config.Config{MediaRoot: root}, testLogger())
	got, err := gen.Generate(context.Background(), &model.ManifestRequest{AssetID: "asset-wv", DRM: media.DRMModeWidevine})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	if string(got) != wantBody {
		t.Fatalf("expected DRM manifest served verbatim (no Inject)\n got: %s\nwant: %s", got, wantBody)
	}
}
