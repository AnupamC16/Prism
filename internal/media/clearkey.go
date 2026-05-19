package media

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const DRMModeClearKey = "clearkey"

type ClearKeyMetadata struct {
	AssetID   string `json:"asset_id"`
	KIDHex    string `json:"kid_hex"`
	KeyHex    string `json:"key_hex"`
	KIDBase64 string `json:"kid_base64url"`
	KeyBase64 string `json:"key_base64url"`
	CreatedAt string `json:"created_at"`
}

func NewClearKeyMetadata(assetID string) (*ClearKeyMetadata, error) {
	kid, err := randomBytes(16)
	if err != nil {
		return nil, fmt.Errorf("generate key id: %w", err)
	}
	key, err := randomBytes(16)
	if err != nil {
		return nil, fmt.Errorf("generate content key: %w", err)
	}

	return &ClearKeyMetadata{
		AssetID:   assetID,
		KIDHex:    hex.EncodeToString(kid),
		KeyHex:    hex.EncodeToString(key),
		KIDBase64: base64.RawURLEncoding.EncodeToString(kid),
		KeyBase64: base64.RawURLEncoding.EncodeToString(key),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func LoadClearKeyMetadata(root, assetID string) (*ClearKeyMetadata, error) {
	body, err := os.ReadFile(ClearKeyMetadataPath(root, assetID))
	if err != nil {
		return nil, err
	}

	var metadata ClearKeyMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return nil, fmt.Errorf("decode ClearKey metadata: %w", err)
	}
	return &metadata, nil
}

func SaveClearKeyMetadata(root string, metadata *ClearKeyMetadata) error {
	path := ClearKeyMetadataPath(root, metadata.AssetID)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create DRM metadata directory: %w", err)
	}

	body, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("encode ClearKey metadata: %w", err)
	}

	if err := os.WriteFile(path, body, 0o600); err != nil {
		return fmt.Errorf("write ClearKey metadata: %w", err)
	}
	return nil
}

func randomBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	return b, err
}
