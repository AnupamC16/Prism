package controller

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/anupam-chopra/prism/internal/api/response"
	"github.com/anupam-chopra/prism/internal/logger"
	"github.com/anupam-chopra/prism/internal/media"
	"github.com/anupam-chopra/prism/internal/model"
	"github.com/anupam-chopra/prism/internal/service"
)

const maxUploadBytes = 2 << 30

type MediaController struct {
	processor    *media.Processor
	mediaRoot    string
	playerDemo   string
	tokenService service.TokenServiceI
	logger       *slog.Logger
	assetIDChars *regexp.Regexp
}

func NewMediaController(processor *media.Processor, mediaRoot, playerDemo string, tokenService service.TokenServiceI, log *slog.Logger) *MediaController {
	if processor == nil {
		panic("processor cannot be nil")
	}
	if mediaRoot == "" {
		panic("mediaRoot cannot be empty")
	}
	if log == nil {
		panic("logger cannot be nil")
	}

	return &MediaController{
		processor:    processor,
		mediaRoot:    mediaRoot,
		playerDemo:   playerDemo,
		tokenService: tokenService,
		logger:       log,
		assetIDChars: regexp.MustCompile(`[^a-zA-Z0-9-_]+`),
	}
}

func (c *MediaController) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		response.BadRequest(w, "multipart form with file is required")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "file field is required")
		return
	}
	defer file.Close()

	assetID := r.FormValue("asset_id")
	if assetID == "" {
		assetID = c.assetIDFromFilename(header.Filename)
	}
	if !model.IsValidAssetID(assetID) {
		response.ValidationError(w, model.NewValidationError("asset_id", "must match ^[a-zA-Z0-9-_]{1,128}$"))
		return
	}

	drm := r.FormValue("drm")
	if drm != "" && drm != media.DRMModeClearKey && drm != media.DRMModeWidevine && drm != media.DRMModeFairPlay {
		response.ValidationError(w, model.NewValidationError("drm", "must be one of clearkey, widevine, fairplay when provided"))
		return
	}

	if err := c.processor.ProcessWithOptions(r.Context(), assetID, file, media.ProcessOptions{DRM: drm}); err != nil {
		response.InternalError(w, c.logger, err, logger.GetRequestID(r.Context()))
		return
	}

	data := map[string]string{
		"asset_id":      assetID,
		"hls_manifest":  "/manifest/hls/" + assetID,
		"dash_manifest": "/manifest/dash/" + assetID,
		"player_url":    "/demo?asset_id=" + assetID,
	}
	switch drm {
	case media.DRMModeClearKey:
		data["clearkey_dash_manifest"] = "/manifest/dash/" + assetID + "?drm=clearkey"
	case media.DRMModeWidevine:
		data["widevine_dash_manifest"] = "/manifest/dash/" + assetID + "?drm=widevine"
	case media.DRMModeFairPlay:
		data["fairplay_hls_manifest"] = "/manifest/hls/" + assetID + "?drm=fairplay"
	}

	response.Success(w, http.StatusCreated, data)
}

func (c *MediaController) ClearKeyLicense(w http.ResponseWriter, r *http.Request) {
	assetID := r.Header.Get("X-Asset-ID")
	if assetID == "" {
		response.BadRequest(w, "X-Asset-ID header is required")
		return
	}
	if !model.IsValidAssetID(assetID) {
		response.ValidationError(w, model.NewValidationError("asset_id", "must match ^[a-zA-Z0-9-_]{1,128}$"))
		return
	}

	token := r.Header.Get("X-DRM-Token")
	if token == "" {
		response.Unauthorized(w, "X-DRM-Token header is required")
		return
	}
	if c.tokenService != nil {
		if _, err := c.tokenService.Validate(r.Context(), token, assetID); err != nil {
			response.LicenseUnauthorized(w, err.Error())
			return
		}
	}

	metadata, err := media.LoadClearKeyMetadata(c.mediaRoot, assetID)
	if err != nil {
		if os.IsNotExist(err) {
			response.NotFound(w, "clearkey metadata", assetID)
			return
		}
		response.InternalError(w, c.logger, err, logger.GetRequestID(r.Context()))
		return
	}

	body := map[string]any{
		"keys": []map[string]string{
			{
				"kty": "oct",
				"kid": metadata.KIDBase64,
				"k":   metadata.KeyBase64,
			},
		},
		"type": "temporary",
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
}

func (c *MediaController) ServeHLSAsset(w http.ResponseWriter, r *http.Request) {
	assetID := r.PathValue("id")
	relPath := r.PathValue("file")
	filePath, err := media.SafeHLSFilePath(c.mediaRoot, assetID, relPath)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			response.NotFound(w, "asset file", filepath.ToSlash(filepath.Join(assetID, "hls", relPath)))
			return
		}
		response.InternalError(w, c.logger, err, logger.GetRequestID(r.Context()))
		return
	}

	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".m3u8":
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	case ".ts":
		w.Header().Set("Content-Type", "video/mp2t")
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, filePath)
}

func (c *MediaController) ServeDASHAsset(w http.ResponseWriter, r *http.Request) {
	assetID := r.PathValue("id")
	relPath := r.PathValue("file")
	filePath, err := media.SafeDASHFilePath(c.mediaRoot, assetID, relPath)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			response.NotFound(w, "asset file", filepath.ToSlash(filepath.Join(assetID, "dash", relPath)))
			return
		}
		response.InternalError(w, c.logger, err, logger.GetRequestID(r.Context()))
		return
	}

	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".mpd":
		w.Header().Set("Content-Type", "application/dash+xml")
	case ".m4s", ".mp4":
		w.Header().Set("Content-Type", "video/iso.segment")
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, filePath)
}

func (c *MediaController) ServeClearKeyDASHAsset(w http.ResponseWriter, r *http.Request) {
	assetID := r.PathValue("id")
	relPath := r.PathValue("file")
	filePath, err := media.SafeClearKeyDASHFilePath(c.mediaRoot, assetID, relPath)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			response.NotFound(w, "asset file", filepath.ToSlash(filepath.Join(assetID, "dash_clearkey", relPath)))
			return
		}
		response.InternalError(w, c.logger, err, logger.GetRequestID(r.Context()))
		return
	}

	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".mpd":
		w.Header().Set("Content-Type", "application/dash+xml")
	case ".m4s", ".mp4":
		w.Header().Set("Content-Type", "video/iso.segment")
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, filePath)
}

func (c *MediaController) ServeDRMDASHAsset(w http.ResponseWriter, r *http.Request) {
	assetID := r.PathValue("id")
	relPath := r.PathValue("file")
	filePath, err := media.SafeDRMDASHFilePath(c.mediaRoot, assetID, relPath)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			response.NotFound(w, "asset file", filepath.ToSlash(filepath.Join(assetID, "dash_drm", relPath)))
			return
		}
		response.InternalError(w, c.logger, err, logger.GetRequestID(r.Context()))
		return
	}

	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".mpd":
		w.Header().Set("Content-Type", "application/dash+xml")
	case ".m4s", ".mp4":
		w.Header().Set("Content-Type", "video/iso.segment")
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, filePath)
}

func (c *MediaController) ServeFairPlayHLSAsset(w http.ResponseWriter, r *http.Request) {
	assetID := r.PathValue("id")
	relPath := r.PathValue("file")
	filePath, err := media.SafeFairPlayHLSFilePath(c.mediaRoot, assetID, relPath)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			response.NotFound(w, "asset file", filepath.ToSlash(filepath.Join(assetID, "hls_fairplay", relPath)))
			return
		}
		response.InternalError(w, c.logger, err, logger.GetRequestID(r.Context()))
		return
	}

	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".m3u8":
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	case ".ts":
		w.Header().Set("Content-Type", "video/mp2t")
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, filePath)
}

func (c *MediaController) Demo(w http.ResponseWriter, r *http.Request) {
	if c.playerDemo == "" {
		response.NotFound(w, "demo", "player")
		return
	}
	http.ServeFile(w, r, c.playerDemo)
}

func (c *MediaController) assetIDFromFilename(filename string) string {
	base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	base = c.assetIDChars.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-_")
	if base == "" {
		base = "asset"
	}
	base = strings.Trim(base, "-_")
	if len(base) > 80 {
		base = base[:80]
	}
	return base + "-" + time.Now().UTC().Format("20060102150405")
}
