package request

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/anupam-chopra/prism/internal/model"
)

type ManifestRequest struct {
	AssetID      string
	Codec        string
	MaxBandwidth int
	Resolution   string
}

func ManifestFromHTTP(r *http.Request) (*ManifestRequest, error) {
	req := &ManifestRequest{
		AssetID:    r.PathValue("id"),
		Codec:      r.URL.Query().Get("codec"),
		Resolution: r.URL.Query().Get("resolution"),
	}

	if raw := r.URL.Query().Get("maxBandwidth"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return nil, model.NewValidationError("maxBandwidth", "must be a positive integer")
		}
		req.MaxBandwidth = v
	}

	return req, nil
}

func (r *ManifestRequest) Validate() error {
	return r.ToDomain().Validate()
}

func (r *ManifestRequest) ToDomain() *model.ManifestRequest {
	return &model.ManifestRequest{
		AssetID:      r.AssetID,
		Codec:        r.Codec,
		MaxBandwidth: r.MaxBandwidth,
		Resolution:   r.Resolution,
	}
}

func (r *ManifestRequest) FilterHash() string {
	return fmt.Sprintf("%s:%d:%s", r.Codec, r.MaxBandwidth, r.Resolution)
}
