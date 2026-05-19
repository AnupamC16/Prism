package cdn

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/anupam-chopra/prism/internal/config"
)

type CloudFront struct {
	enabled    bool
	domain     string
	keyPairID  string
	privateKey *rsa.PrivateKey
	logger     *slog.Logger
}

func NewCloudFront(cfg *config.Config, logger *slog.Logger) (*CloudFront, error) {
	if cfg.CloudFrontDomain == "" && cfg.CloudFrontKeyPairID == "" && cfg.CloudFrontPrivateKey == "" {
		if logger != nil {
			logger.Warn("cloudfront signing disabled; manifest URIs will not be rewritten")
		}
		return &CloudFront{logger: logger}, nil
	}

	pemData := strings.ReplaceAll(cfg.CloudFrontPrivateKey, `\n`, "\n")
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, errors.New("invalid PEM block")
	}

	var rsaKey *rsa.PrivateKey

	pkcs1Key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		rsaKey = pkcs1Key
	} else {
		pkcs8Key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("parse private key: %w", err2)
		}
		var ok bool
		rsaKey, ok = pkcs8Key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("PKCS8 key is not an RSA private key")
		}
	}

	return &CloudFront{
		enabled:    true,
		domain:     cfg.CloudFrontDomain,
		keyPairID:  cfg.CloudFrontKeyPairID,
		privateKey: rsaKey,
		logger:     logger,
	}, nil
}

func (c *CloudFront) SignURL(originalURL string, expires time.Time) (string, error) {
	if !c.enabled {
		return originalURL, nil
	}

	policy := fmt.Sprintf(
		`{"Statement":[{"Resource":"%s","Condition":{"DateLessThan":{"AWS:EpochTime":%d}}}]}`,
		originalURL,
		expires.Unix(),
	)

	hash := sha1.Sum([]byte(policy))
	sig, err := rsa.SignPKCS1v15(nil, c.privateKey, crypto.SHA1, hash[:])
	if err != nil {
		return "", fmt.Errorf("sign CloudFront policy: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(sig)
	encoded = strings.NewReplacer("+", "-", "=", "_", "/", "~").Replace(encoded)

	signed := fmt.Sprintf("%s?Expires=%d&Signature=%s&Key-Pair-Id=%s",
		originalURL, expires.Unix(), encoded, c.keyPairID)
	return signed, nil
}

func (c *CloudFront) RewriteManifestURIs(ctx context.Context, manifest []byte, assetID string) ([]byte, error) {
	if !c.enabled {
		return manifest, nil
	}

	expiry := time.Now().UTC().Add(24 * time.Hour)
	result := string(manifest)
	lines := strings.Split(result, "\n")

	for _, line := range lines {
		uri := strings.TrimSpace(line)
		if uri == "" ||
			strings.HasPrefix(uri, "#") ||
			strings.HasPrefix(uri, "<") ||
			strings.ContainsAny(uri, " \t") ||
			strings.HasPrefix(uri, "/assets/") ||
			strings.HasPrefix(uri, "http://") ||
			strings.HasPrefix(uri, "https://") ||
			strings.HasPrefix(uri, "data:") ||
			strings.HasPrefix(uri, "skd://") {
			continue
		}

		relativePath := strings.TrimPrefix(uri, "/")
		fullURL := fmt.Sprintf("https://%s/assets/%s/%s", c.domain, assetID, relativePath)

		signed, err := c.SignURL(fullURL, expiry)
		if err != nil {
			if c.logger != nil {
				c.logger.WarnContext(ctx, "failed to sign manifest URI", "uri", uri, "error", err)
			}
			result = strings.Replace(result, uri, fullURL, 1)
			continue
		}
		result = strings.Replace(result, uri, signed, 1)
	}

	return []byte(result), nil
}
