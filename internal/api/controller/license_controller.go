package controller

import (
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/anupam-chopra/prism/internal/api/response"
	"github.com/anupam-chopra/prism/internal/logger"
	"github.com/anupam-chopra/prism/internal/model"
	"github.com/anupam-chopra/prism/internal/service"
)

type LicenseController struct {
	service     service.DRMServiceI
	auditLogger *slog.Logger
	logger      *slog.Logger
}

func NewLicenseController(service service.DRMServiceI, auditLogger *slog.Logger, logger *slog.Logger) *LicenseController {
	if service == nil {
		panic("service cannot be nil")
	}
	if auditLogger == nil {
		panic("auditLogger cannot be nil")
	}
	if logger == nil {
		panic("logger cannot be nil")
	}

	return &LicenseController{
		service:     service,
		auditLogger: auditLogger,
		logger:      logger,
	}
}

func (c *LicenseController) handleLicense(w http.ResponseWriter, r *http.Request, drmType string) {
	const maxChallengeBodyBytes = 1 << 20

	body, err := io.ReadAll(io.LimitReader(r.Body, maxChallengeBodyBytes+1))
	if err != nil {
		response.InternalError(w, c.logger, err, logger.GetRequestID(r.Context()))
		return
	}
	if len(body) > maxChallengeBodyBytes {
		response.Error(w, http.StatusRequestEntityTooLarge, response.ErrCodeValidation, "challenge body exceeds 1MB limit")
		return
	}
	if len(body) == 0 {
		response.BadRequest(w, "challenge body is required")
		return
	}

	token := r.Header.Get("X-DRM-Token")
	if token == "" {
		response.Unauthorized(w, "X-DRM-Token header is required")
		return
	}

	assetID := r.Header.Get("X-Asset-ID")
	if assetID == "" {
		response.BadRequest(w, "X-Asset-ID header is required")
		return
	}

	req := &model.LicenseRequest{
		DRMType:   drmType,
		Challenge: body,
		Token:     token,
		AssetID:   assetID,
	}

	if drmType == model.DRMTypeFairPlay {
		spcBase64 := r.Header.Get("X-FairPlay-SPC")
		if spcBase64 != "" {
			spcBytes, err := base64.StdEncoding.DecodeString(spcBase64)
			if err != nil {
				response.BadRequest(w, "invalid X-FairPlay-SPC header format")
				return
			}
			req.SPCBytes = spcBytes
		}
	}

	if err := req.Validate(); err != nil {
		response.ValidationError(w, err)
		return
	}

	start := time.Now()
	licenseBytes, err := c.service.GetLicense(r.Context(), req)

	c.auditLogger.Info("license_request",
		"drm_type", drmType,
		"asset_id", assetID,
		"success", err == nil,
		"duration_ms", time.Since(start).Milliseconds(),
		"request_id", logger.GetRequestID(r.Context()),
	)

	if err != nil {
		if model.IsTokenError(err) {
			response.LicenseUnauthorized(w, err.Error())
			return
		}
		if model.IsUpstreamError(err) {
			response.LicenseBadGateway(w, drmType)
			return
		}
		response.InternalError(w, c.logger, err, logger.GetRequestID(r.Context()))
		return
	}

	response.LicenseBytes(w, drmType, licenseBytes)
}

func (c *LicenseController) Widevine(w http.ResponseWriter, r *http.Request) {
	c.handleLicense(w, r, model.DRMTypeWidevine)
}

func (c *LicenseController) FairPlay(w http.ResponseWriter, r *http.Request) {
	c.handleLicense(w, r, model.DRMTypeFairPlay)
}

func (c *LicenseController) PlayReady(w http.ResponseWriter, r *http.Request) {
	c.handleLicense(w, r, model.DRMTypePlayReady)
}
