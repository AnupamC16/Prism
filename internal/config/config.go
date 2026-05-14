package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration parameters for the Prism service.
type Config struct {
	// Port is the server listening port (env: PORT, default: ":8080").
	Port string
	// LogLevel sets the logging verbosity (env: LOG_LEVEL, default: "info").
	LogLevel string
	// RedisURL is the connection string for Redis (env: REDIS_URL, required).
	RedisURL string
	// JWTSecret is the secret key for signing JWTs (env: JWT_SECRET, required, minimum 32 characters).
	JWTSecret string
	// TokenTTLSeconds is the time-to-live for authentication tokens in seconds (env: TOKEN_TTL_SECONDS, default: 3600).
	TokenTTLSeconds int
	// WidevineURL is the endpoint for Widevine license requests (env: WIDEVINE_URL, required).
	WidevineURL string
	// WidevineAPIKey is the API key for Widevine authentication (env: WIDEVINE_API_KEY, required).
	WidevineAPIKey string
	// FairPlayURL is the endpoint for FairPlay license requests (env: FAIRPLAY_URL, required).
	FairPlayURL string
	// FairPlayCertURL is the URL to fetch the FairPlay certificate (env: FAIRPLAY_CERT_URL, required).
	FairPlayCertURL string
	// FairPlaySecret is the secret used for FairPlay requests (env: FAIRPLAY_SECRET, required).
	FairPlaySecret string
	// PlayReadyURL is the endpoint for PlayReady license requests (env: PLAYREADY_URL, required).
	PlayReadyURL string
	// CloudFrontDomain is the domain used for CloudFront CDN (env: CLOUDFRONT_DOMAIN, required).
	CloudFrontDomain string
	// CloudFrontKeyPairID is the key pair ID for CloudFront signed URLs/cookies (env: CLOUDFRONT_KEY_PAIR_ID, required).
	CloudFrontKeyPairID string
	// CloudFrontPrivateKey is the PEM formatted private key for CloudFront (env: CLOUDFRONT_PRIVATE_KEY, required).
	CloudFrontPrivateKey string
	// ManifestCacheTTL is the cache duration for manifests in seconds (env: MANIFEST_CACHE_TTL, default: 30).
	ManifestCacheTTL int
	// LicenseCacheTTL is the cache duration for DRM licenses in seconds (env: LICENSE_CACHE_TTL, default: 300).
	LicenseCacheTTL int
	// CertCacheTTL is the cache duration for DRM certificates in seconds (env: CERT_CACHE_TTL, default: 3600).
	CertCacheTTL int
	// Version is the current version of the application (env: VERSION, default: "1.0.0").
	Version string
}

// Load reads configuration from environment variables, applying defaults
// and validating required fields and constraints. It returns a fully populated Config.
func Load() (*Config, error) {
	c := &Config{
		Port:                 getEnvOrDefault("PORT", ":8080"),
		LogLevel:             getEnvOrDefault("LOG_LEVEL", "info"),
		RedisURL:             os.Getenv("REDIS_URL"),
		JWTSecret:            os.Getenv("JWT_SECRET"),
		WidevineURL:          os.Getenv("WIDEVINE_URL"),
		WidevineAPIKey:       os.Getenv("WIDEVINE_API_KEY"),
		FairPlayURL:          os.Getenv("FAIRPLAY_URL"),
		FairPlayCertURL:      os.Getenv("FAIRPLAY_CERT_URL"),
		FairPlaySecret:       os.Getenv("FAIRPLAY_SECRET"),
		PlayReadyURL:         os.Getenv("PLAYREADY_URL"),
		CloudFrontDomain:     os.Getenv("CLOUDFRONT_DOMAIN"),
		CloudFrontKeyPairID:  os.Getenv("CLOUDFRONT_KEY_PAIR_ID"),
		CloudFrontPrivateKey: os.Getenv("CLOUDFRONT_PRIVATE_KEY"),
		Version:              getEnvOrDefault("VERSION", "1.0.0"),
	}

	var missingVars []string

	if c.RedisURL == "" {
		missingVars = append(missingVars, "REDIS_URL")
	}
	if c.JWTSecret == "" {
		missingVars = append(missingVars, "JWT_SECRET")
	}
	if c.WidevineURL == "" {
		missingVars = append(missingVars, "WIDEVINE_URL")
	}
	if c.WidevineAPIKey == "" {
		missingVars = append(missingVars, "WIDEVINE_API_KEY")
	}
	if c.FairPlayURL == "" {
		missingVars = append(missingVars, "FAIRPLAY_URL")
	}
	if c.FairPlayCertURL == "" {
		missingVars = append(missingVars, "FAIRPLAY_CERT_URL")
	}
	if c.FairPlaySecret == "" {
		missingVars = append(missingVars, "FAIRPLAY_SECRET")
	}
	if c.PlayReadyURL == "" {
		missingVars = append(missingVars, "PLAYREADY_URL")
	}
	if c.CloudFrontDomain == "" {
		missingVars = append(missingVars, "CLOUDFRONT_DOMAIN")
	}
	if c.CloudFrontKeyPairID == "" {
		missingVars = append(missingVars, "CLOUDFRONT_KEY_PAIR_ID")
	}
	if c.CloudFrontPrivateKey == "" {
		missingVars = append(missingVars, "CLOUDFRONT_PRIVATE_KEY")
	}

	if len(missingVars) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missingVars, ", "))
	}

	var err error

	c.TokenTTLSeconds, err = parseEnvInt("TOKEN_TTL_SECONDS", 3600)
	if err != nil {
		return nil, err
	}

	c.ManifestCacheTTL, err = parseEnvInt("MANIFEST_CACHE_TTL", 30)
	if err != nil {
		return nil, err
	}

	c.LicenseCacheTTL, err = parseEnvInt("LICENSE_CACHE_TTL", 300)
	if err != nil {
		return nil, err
	}

	c.CertCacheTTL, err = parseEnvInt("CERT_CACHE_TTL", 3600)
	if err != nil {
		return nil, err
	}

	if len(c.JWTSecret) < 32 {
		return nil, errors.New("JWT_SECRET must be at least 32 characters long")
	}

	if c.TokenTTLSeconds < 60 || c.TokenTTLSeconds > 86400 {
		return nil, errors.New("TOKEN_TTL_SECONDS must be between 60 and 86400")
	}

	return c, nil
}

// MustLoad calls Load() and exits the application using os.Exit(1) if an error occurs.
func MustLoad() *Config {
	c, err := Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}
	return c
}

// Validate re-runs all validation rules on the Config struct.
// It returns nil if all fields are valid, or a descriptive error if any rule fails.
func (c *Config) Validate() error {
	var missingVars []string

	if c.RedisURL == "" {
		missingVars = append(missingVars, "REDIS_URL")
	}
	if c.JWTSecret == "" {
		missingVars = append(missingVars, "JWT_SECRET")
	}
	if c.WidevineURL == "" {
		missingVars = append(missingVars, "WIDEVINE_URL")
	}
	if c.WidevineAPIKey == "" {
		missingVars = append(missingVars, "WIDEVINE_API_KEY")
	}
	if c.FairPlayURL == "" {
		missingVars = append(missingVars, "FAIRPLAY_URL")
	}
	if c.FairPlayCertURL == "" {
		missingVars = append(missingVars, "FAIRPLAY_CERT_URL")
	}
	if c.FairPlaySecret == "" {
		missingVars = append(missingVars, "FAIRPLAY_SECRET")
	}
	if c.PlayReadyURL == "" {
		missingVars = append(missingVars, "PLAYREADY_URL")
	}
	if c.CloudFrontDomain == "" {
		missingVars = append(missingVars, "CLOUDFRONT_DOMAIN")
	}
	if c.CloudFrontKeyPairID == "" {
		missingVars = append(missingVars, "CLOUDFRONT_KEY_PAIR_ID")
	}
	if c.CloudFrontPrivateKey == "" {
		missingVars = append(missingVars, "CLOUDFRONT_PRIVATE_KEY")
	}

	if len(missingVars) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missingVars, ", "))
	}

	if len(c.JWTSecret) < 32 {
		return errors.New("JWT_SECRET must be at least 32 characters long")
	}

	if c.TokenTTLSeconds < 60 || c.TokenTTLSeconds > 86400 {
		return errors.New("TOKEN_TTL_SECONDS must be between 60 and 86400")
	}

	return nil
}

// getEnvOrDefault returns the value of an environment variable or a fallback default.
func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// parseEnvInt reads an environment variable and parses it as an int.
// If the variable is empty, it returns the provided default value.
// It returns a descriptive error naming the field if the parsing fails.
func parseEnvInt(key string, defaultValue int) (int, error) {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue, nil
	}

	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse environment variable %s as int: %v", key, err)
	}

	return val, nil
}
