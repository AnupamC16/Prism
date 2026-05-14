package cache

import (
	"errors"
	"fmt"
	"strings"
)

const (
	manifestKeyFmt = "prism:manifest:%s:%s:%s" // type:assetID:filterHash
	licenseKeyFmt  = "prism:license:%s:%s"     // drmType:challengeHash
	tokenKeyFmt    = "prism:token:%s"          // jti
	certKeyFmt     = "prism:cert:%s"           // drmType
	assetKeyFmt    = "prism:asset:%s"          // assetID
)

func ManifestKey(manifestType, assetID, filterHash string) string {
	return fmt.Sprintf(manifestKeyFmt, manifestType, assetID, filterHash)
}

func LicenseKey(drmType, challengeHash string) string {
	return fmt.Sprintf(licenseKeyFmt, drmType, challengeHash)
}

func TokenKey(jti string) string {
	return fmt.Sprintf(tokenKeyFmt, jti)
}

func CertKey(drmType string) string {
	return fmt.Sprintf(certKeyFmt, drmType)
}

func AssetKey(assetID string) string {
	return fmt.Sprintf(assetKeyFmt, assetID)
}

// ParseManifestKey parses a manifest cache key back into its components.
// Expected format: prism:manifest:<manifestType>:<assetID>:<filterHash>
func ParseManifestKey(key string) (manifestType, assetID, filterHash string, err error) {
	const prefix = "prism:manifest:"
	if !strings.HasPrefix(key, prefix) {
		return "", "", "", errors.New("cache: key is not a manifest key")
	}
	rest := key[len(prefix):]
	parts := strings.SplitN(rest, ":", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("cache: manifest key has wrong format: %q", key)
	}
	if parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("cache: manifest key contains empty segment: %q", key)
	}
	return parts[0], parts[1], parts[2], nil
}

// ValidateKey returns an error if key is empty, contains whitespace, or exceeds 512 bytes.
func ValidateKey(key string) error {
	if key == "" {
		return errors.New("cache: key must not be empty")
	}
	if strings.ContainsAny(key, " \t\n\r\v\f") {
		return errors.New("cache: key must not contain whitespace")
	}
	if len(key) > 512 {
		return errors.New("cache: key exceeds maximum length of 512 bytes")
	}
	return nil
}
