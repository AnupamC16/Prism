package controller

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/anupam-chopra/prism/internal/api/response"
	"github.com/anupam-chopra/prism/internal/logger"
	"github.com/anupam-chopra/prism/internal/model"
	"github.com/anupam-chopra/prism/internal/service"
)

type ManifestController struct {
	service service.ManifestServiceI
	logger  *slog.Logger
}

func NewManifestController(service service.ManifestServiceI, log *slog.Logger) *ManifestController {
	if service == nil {
		panic("ManifestService is required")
	}
	if log == nil {
		panic("Logger is required")
	}
	return &ManifestController{
		service: service,
		logger:  log,
	}
}

func (c *ManifestController) GetHLS(w http.ResponseWriter, r *http.Request) {
	assetID := r.PathValue("id")
	if assetID == "" {
		response.BadRequest(w, "asset id is required")
		return
	}

	req := &model.ManifestRequest{
		AssetID:    assetID,
		Codec:      r.URL.Query().Get("codec"),
		Resolution: r.URL.Query().Get("resolution"),
		DRM:        r.URL.Query().Get("drm"),
	}

	maxBandwidthStr := r.URL.Query().Get("maxBandwidth")
	if maxBandwidthStr != "" {
		maxBw, err := strconv.Atoi(maxBandwidthStr)
		if err != nil {
			response.BadRequest(w, "invalid maxBandwidth parameter")
			return
		}
		req.MaxBandwidth = maxBw
	}

	if err := req.Validate(); err != nil {
		response.ValidationError(w, err)
		return
	}

	manifestBytes, err := c.service.GetHLS(r.Context(), req)
	if err != nil {
		if model.IsNotFoundError(err) {
			response.ManifestNotFound(w, assetID)
			return
		}
		response.InternalError(w, c.logger, err, logger.GetRequestID(r.Context()))
		return
	}

	response.HLSManifest(w, manifestBytes)
}

func (c *ManifestController) GetDASH(w http.ResponseWriter, r *http.Request) {
	assetID := r.PathValue("id")
	if assetID == "" {
		response.BadRequest(w, "asset id is required")
		return
	}

	req := &model.ManifestRequest{
		AssetID:    assetID,
		Codec:      r.URL.Query().Get("codec"),
		Resolution: r.URL.Query().Get("resolution"),
		DRM:        r.URL.Query().Get("drm"),
	}

	maxBandwidthStr := r.URL.Query().Get("maxBandwidth")
	if maxBandwidthStr != "" {
		maxBw, err := strconv.Atoi(maxBandwidthStr)
		if err != nil {
			response.BadRequest(w, "invalid maxBandwidth parameter")
			return
		}
		req.MaxBandwidth = maxBw
	}

	if err := req.Validate(); err != nil {
		response.ValidationError(w, err)
		return
	}

	manifestBytes, err := c.service.GetDASH(r.Context(), req)
	if err != nil {
		if model.IsNotFoundError(err) {
			response.ManifestNotFound(w, assetID)
			return
		}
		response.InternalError(w, c.logger, err, logger.GetRequestID(r.Context()))
		return
	}

	response.DASHManifest(w, manifestBytes)
}
