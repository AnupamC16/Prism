package media

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/anupam-chopra/prism/internal/model"
)

func HLSManifestPath(root, assetID string) string {
	return filepath.Join(root, assetID, "hls", "index.m3u8")
}

func DASHManifestPath(root, assetID string) string {
	return filepath.Join(root, assetID, "dash", "index.mpd")
}

func ClearKeyDASHManifestPath(root, assetID string) string {
	return filepath.Join(root, assetID, "dash_clearkey", "index.mpd")
}

func ClearKeyMetadataPath(root, assetID string) string {
	return filepath.Join(root, assetID, "drm", "clearkey.json")
}

func DRMMetadataPath(root, assetID string) string {
	return filepath.Join(root, assetID, "drm", "widevine.json")
}

func DRMDASHManifestPath(root, assetID string) string {
	return filepath.Join(root, assetID, "dash_drm", "index.mpd")
}

func FairPlayMetadataPath(root, assetID string) string {
	return filepath.Join(root, assetID, "drm", "fairplay.json")
}

func FairPlayHLSManifestPath(root, assetID string) string {
	return filepath.Join(root, assetID, "hls_fairplay", "index.m3u8")
}

func HasLocalHLS(root, assetID string) bool {
	return hasLocalManifest(HLSManifestPath(root, assetID), root, assetID)
}

func HasLocalDASH(root, assetID string) bool {
	return hasLocalManifest(DASHManifestPath(root, assetID), root, assetID)
}

func HasLocalClearKeyDASH(root, assetID string) bool {
	return hasLocalManifest(ClearKeyDASHManifestPath(root, assetID), root, assetID)
}

func HasLocalDRMDASH(root, assetID string) bool {
	return hasLocalManifest(DRMDASHManifestPath(root, assetID), root, assetID)
}

func HasLocalFairPlayHLS(root, assetID string) bool {
	return hasLocalManifest(FairPlayHLSManifestPath(root, assetID), root, assetID)
}

func hasLocalManifest(path, root, assetID string) bool {
	if root == "" || !model.IsValidAssetID(assetID) {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func SafeHLSFilePath(root, assetID, relPath string) (string, error) {
	return safeAssetFilePath(root, assetID, "hls", relPath)
}

func SafeDASHFilePath(root, assetID, relPath string) (string, error) {
	return safeAssetFilePath(root, assetID, "dash", relPath)
}

func SafeClearKeyDASHFilePath(root, assetID, relPath string) (string, error) {
	return safeAssetFilePath(root, assetID, "dash_clearkey", relPath)
}

func SafeDRMDASHFilePath(root, assetID, relPath string) (string, error) {
	return safeAssetFilePath(root, assetID, "dash_drm", relPath)
}

func SafeFairPlayHLSFilePath(root, assetID, relPath string) (string, error) {
	return safeAssetFilePath(root, assetID, "hls_fairplay", relPath)
}

func safeAssetFilePath(root, assetID, mediaType, relPath string) (string, error) {
	if root == "" {
		return "", errors.New("media root is not configured")
	}
	if !model.IsValidAssetID(assetID) {
		return "", errors.New("invalid asset id")
	}

	cleaned := filepath.Clean(filepath.FromSlash(relPath))
	if cleaned == "." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) || cleaned == ".." || filepath.IsAbs(cleaned) {
		return "", errors.New("invalid asset file path")
	}

	return filepath.Join(root, assetID, mediaType, cleaned), nil
}
