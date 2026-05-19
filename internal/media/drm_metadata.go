package media

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const DRMModeWidevine = "widevine"

type DRMMetadata struct {
	AssetID   string `json:"asset_id"`
	KIDHex    string `json:"kid_hex"`
	KeyHex    string `json:"key_hex"`
	KIDBase64 string `json:"kid_base64url"`
	KeyBase64 string `json:"key_base64url"`
	CreatedAt string `json:"created_at"`
}

func NewDRMMetadata(assetID string) (*DRMMetadata, error) {
	kid, err := randomBytes(16)
	if err != nil {
		return nil, fmt.Errorf("generate key id: %w", err)
	}
	key, err := randomBytes(16)
	if err != nil {
		return nil, fmt.Errorf("generate content key: %w", err)
	}

	return &DRMMetadata{
		AssetID:   assetID,
		KIDHex:    hex.EncodeToString(kid),
		KeyHex:    hex.EncodeToString(key),
		KIDBase64: base64.RawURLEncoding.EncodeToString(kid),
		KeyBase64: base64.RawURLEncoding.EncodeToString(key),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func LoadDRMMetadata(root, assetID string) (*DRMMetadata, error) {
	body, err := os.ReadFile(DRMMetadataPath(root, assetID))
	if err != nil {
		return nil, err
	}

	var metadata DRMMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return nil, fmt.Errorf("decode DRM metadata: %w", err)
	}
	return &metadata, nil
}

func SaveDRMMetadata(root string, metadata *DRMMetadata) error {
	path := DRMMetadataPath(root, metadata.AssetID)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create DRM metadata directory: %w", err)
	}

	body, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("encode DRM metadata: %w", err)
	}

	if err := os.WriteFile(path, body, 0o600); err != nil {
		return fmt.Errorf("write DRM metadata: %w", err)
	}
	return nil
}
