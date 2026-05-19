package media

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestNewDRMMetadata_GeneratesRandom16ByteKeyAndKID(t *testing.T) {
	meta, err := NewDRMMetadata("asset-1")
	if err != nil {
		t.Fatalf("NewDRMMetadata returned error: %v", err)
	}

	if meta.AssetID != "asset-1" {
		t.Fatalf("AssetID mismatch: got %q want %q", meta.AssetID, "asset-1")
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
	if meta.CreatedAt == "" {
		t.Fatalf("CreatedAt must be populated")
	}
}

func TestNewDRMMetadata_DistinctKeyAndKIDPerCall(t *testing.T) {
	a, err := NewDRMMetadata("asset-1")
	if err != nil {
		t.Fatalf("NewDRMMetadata returned error: %v", err)
	}
	b, err := NewDRMMetadata("asset-1")
	if err != nil {
		t.Fatalf("NewDRMMetadata returned error: %v", err)
	}
	if a.KIDHex == b.KIDHex {
		t.Fatalf("expected distinct KIDs across calls, got duplicate %q", a.KIDHex)
	}
	if a.KeyHex == b.KeyHex {
		t.Fatalf("expected distinct keys across calls, got duplicate %q", a.KeyHex)
	}
}

func TestSaveAndLoadDRMMetadata_RoundTrip(t *testing.T) {
	root := t.TempDir()
	meta, err := NewDRMMetadata("asset-rt")
	if err != nil {
		t.Fatalf("NewDRMMetadata returned error: %v", err)
	}

	if err := SaveDRMMetadata(root, meta); err != nil {
		t.Fatalf("SaveDRMMetadata returned error: %v", err)
	}

	path := DRMMetadataPath(root, "asset-rt")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected file at %q: %v", path, err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected file mode 0600, got %o", info.Mode().Perm())
	}

	loaded, err := LoadDRMMetadata(root, "asset-rt")
	if err != nil {
		t.Fatalf("LoadDRMMetadata returned error: %v", err)
	}
	if loaded.KIDHex != meta.KIDHex {
		t.Fatalf("KID round-trip mismatch: got %q want %q", loaded.KIDHex, meta.KIDHex)
	}
	if loaded.KeyHex != meta.KeyHex {
		t.Fatalf("Key round-trip mismatch: got %q want %q", loaded.KeyHex, meta.KeyHex)
	}
	if loaded.AssetID != "asset-rt" {
		t.Fatalf("AssetID round-trip mismatch: got %q want %q", loaded.AssetID, "asset-rt")
	}
}

func TestDRMDASHManifestPath(t *testing.T) {
	got := DRMDASHManifestPath("/data/assets", "asset-1")
	want := filepath.Join("/data/assets", "asset-1", "dash_drm", "index.mpd")
	if got != want {
		t.Fatalf("DRMDASHManifestPath: got %q want %q", got, want)
	}
}

func TestDRMMetadataPath(t *testing.T) {
	got := DRMMetadataPath("/data/assets", "asset-1")
	want := filepath.Join("/data/assets", "asset-1", "drm", "widevine.json")
	if got != want {
		t.Fatalf("DRMMetadataPath: got %q want %q", got, want)
	}
}

func TestHasLocalDRMDASH(t *testing.T) {
	root := t.TempDir()
	if HasLocalDRMDASH(root, "asset-1") {
		t.Fatalf("expected HasLocalDRMDASH=false when manifest missing")
	}

	manifestPath := DRMDASHManifestPath(root, "asset-1")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte("<MPD/>"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if !HasLocalDRMDASH(root, "asset-1") {
		t.Fatalf("expected HasLocalDRMDASH=true after creating manifest")
	}
}

func TestSafeDRMDASHFilePath_RejectsTraversal(t *testing.T) {
	_, err := SafeDRMDASHFilePath("/data/assets", "asset-1", "../escape.mpd")
	if err == nil {
		t.Fatalf("expected SafeDRMDASHFilePath to reject path traversal")
	}
}
