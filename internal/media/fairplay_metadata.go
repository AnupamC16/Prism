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

const DRMModeFairPlay = "fairplay"

type FairPlayMetadata struct {
	AssetID   string `json:"asset_id"`
	KIDHex    string `json:"kid_hex"`
	KeyHex    string `json:"key_hex"`
	KIDBase64 string `json:"kid_base64url"`
	KeyBase64 string `json:"key_base64url"`
	CreatedAt string `json:"created_at"`
}

func NewFairPlayMetadata(assetID string) (*FairPlayMetadata, error) {
	kid, err := randomBytes(16)
	if err != nil {
		return nil, fmt.Errorf("generate key id: %w", err)
	}
	key, err := randomBytes(16)
	if err != nil {
		return nil, fmt.Errorf("generate content key: %w", err)
	}

	return &FairPlayMetadata{
		AssetID:   assetID,
		KIDHex:    hex.EncodeToString(kid),
		KeyHex:    hex.EncodeToString(key),
		KIDBase64: base64.RawURLEncoding.EncodeToString(kid),
		KeyBase64: base64.RawURLEncoding.EncodeToString(key),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func LoadFairPlayMetadata(root, assetID string) (*FairPlayMetadata, error) {
	body, err := os.ReadFile(FairPlayMetadataPath(root, assetID))
	if err != nil {
		return nil, err
	}

	var metadata FairPlayMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return nil, fmt.Errorf("decode FairPlay metadata: %w", err)
	}
	return &metadata, nil
}

func SaveFairPlayMetadata(root string, metadata *FairPlayMetadata) error {
	path := FairPlayMetadataPath(root, metadata.AssetID)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create FairPlay metadata directory: %w", err)
	}

	body, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("encode FairPlay metadata: %w", err)
	}

	if err := os.WriteFile(path, body, 0o600); err != nil {
		return fmt.Errorf("write FairPlay metadata: %w", err)
	}
	return nil
}
