package request

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"

	"github.com/anupam-chopra/prism/internal/model"
)

type LicenseRequest struct {
	DRMType   string
	Challenge []byte
	Token     string
	AssetID   string
	SPCBytes  []byte
}

func FromHTTP(r *http.Request, drmType string, body []byte) (*LicenseRequest, error) {
	req := &LicenseRequest{
		DRMType:   drmType,
		Challenge: body,
		Token:     r.Header.Get("X-DRM-Token"),
		AssetID:   r.Header.Get("X-Asset-ID"),
	}

	if drmType == "fairplay" {
		raw := r.Header.Get("X-FairPlay-SPC")
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, model.NewValidationError("X-FairPlay-SPC", "must be valid base64")
		}
		req.SPCBytes = decoded
	}

	return req, nil
}

func (r *LicenseRequest) Validate() error {
	return r.ToDomain().Validate()
}

func (r *LicenseRequest) ToDomain() *model.LicenseRequest {
	return &model.LicenseRequest{
		DRMType:   r.DRMType,
		Challenge: r.Challenge,
		Token:     r.Token,
		AssetID:   r.AssetID,
		SPCBytes:  r.SPCBytes,
	}
}

func (r *LicenseRequest) ChallengeHash() string {
	sum := sha256.Sum256(r.Challenge)
	return hex.EncodeToString(sum[:])
}
