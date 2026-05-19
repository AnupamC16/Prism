package media

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestNewFairPlayMetadata_GeneratesRandom16ByteKeyAndKID(t *testing.T) {
	meta, err := NewFairPlayMetadata("asset-fp")
	if err != nil {
		t.Fatalf("NewFairPlayMetadata returned error: %v", err)
	}

	if meta.AssetID != "asset-fp" {
		t.Fatalf("AssetID mismatch: got %q want %q", meta.AssetID, "asset-fp")
	}

	kid, err := hex.DecodeString(meta.KIDHex)
	if err != nil {
		t.Fatalf("KIDHex is not valid hex: %v", err)
	}
	if len(kid) != 16 {
		t.Fatalf("KID must be 16 bytes, got %d", len(kid))
	}

	key, err := hex.DecodeString(meta.KeyHex)
	if err != nil {
		t.Fatalf("KeyHex is not valid hex: %v", err)
	}
	if len(key) != 16 {
		t.Fatalf("Key must be 16 bytes, got %d", len(key))
	}

	if meta.KIDBase64 == "" || meta.KeyBase64 == "" {
		t.Fatalf("base64 encodings must be populated")
	}
}

func TestNewFairPlayMetadata_DistinctKeyAndKIDPerCall(t *testing.T) {
	a, err := NewFairPlayMetadata("asset-fp")
	if err != nil {
		t.Fatalf("NewFairPlayMetadata returned error: %v", err)
	}
	b, err := NewFairPlayMetadata("asset-fp")
	if err != nil {
		t.Fatalf("NewFairPlayMetadata returned error: %v", err)
	}
	if a.KIDHex == b.KIDHex {
		t.Fatalf("expected distinct KIDs across calls")
	}
	if a.KeyHex == b.KeyHex {
		t.Fatalf("expected distinct keys across calls")
	}
}

func TestSaveAndLoadFairPlayMetadata_RoundTrip(t *testing.T) {
	root := t.TempDir()
	meta, err := NewFairPlayMetadata("asset-fp-rt")
	if err != nil {
		t.Fatalf("NewFairPlayMetadata returned error: %v", err)
	}

	if err := SaveFairPlayMetadata(root, meta); err != nil {
		t.Fatalf("SaveFairPlayMetadata returned error: %v", err)
	}

	path := FairPlayMetadataPath(root, "asset-fp-rt")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected file at %q: %v", path, err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected file mode 0600, got %o", info.Mode().Perm())
	}

	loaded, err := LoadFairPlayMetadata(root, "asset-fp-rt")
	if err != nil {
		t.Fatalf("LoadFairPlayMetadata returned error: %v", err)
	}
	if loaded.KIDHex != meta.KIDHex {
		t.Fatalf("KID round-trip mismatch")
	}
	if loaded.KeyHex != meta.KeyHex {
		t.Fatalf("Key round-trip mismatch")
	}
	if loaded.AssetID != "asset-fp-rt" {
		t.Fatalf("AssetID round-trip mismatch")
	}
}

func TestFairPlayHLSManifestPath(t *testing.T) {
	got := FairPlayHLSManifestPath("/data/assets", "asset-1")
	want := filepath.Join("/data/assets", "asset-1", "hls_fairplay", "index.m3u8")
	if got != want {
		t.Fatalf("FairPlayHLSManifestPath: got %q want %q", got, want)
	}
}

func TestFairPlayMetadataPath(t *testing.T) {
	got := FairPlayMetadataPath("/data/assets", "asset-1")
	want := filepath.Join("/data/assets", "asset-1", "drm", "fairplay.json")
	if got != want {
		t.Fatalf("FairPlayMetadataPath: got %q want %q", got, want)
	}
}

func TestHasLocalFairPlayHLS(t *testing.T) {
	root := t.TempDir()
	if HasLocalFairPlayHLS(root, "asset-fp") {
		t.Fatalf("expected HasLocalFairPlayHLS=false when manifest missing")
	}

	manifestPath := FairPlayHLSManifestPath(root, "asset-fp")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte("#EXTM3U"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if !HasLocalFairPlayHLS(root, "asset-fp") {
		t.Fatalf("expected HasLocalFairPlayHLS=true after creating manifest")
	}
}

func TestSafeFairPlayHLSFilePath_RejectsTraversal(t *testing.T) {
	_, err := SafeFairPlayHLSFilePath("/data/assets", "asset-fp", "../escape.m3u8")
	if err == nil {
		t.Fatalf("expected SafeFairPlayHLSFilePath to reject path traversal")
	}
}
