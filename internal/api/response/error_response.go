package response

import (
	"fmt"
	"net/http"
)

func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, ErrCodeValidation, message)
}

func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnauthorized, ErrCodeUnauthorized, message)
}

func NotFound(w http.ResponseWriter, resource, id string) {
	Error(w, http.StatusNotFound, ErrCodeNotFound, fmt.Sprintf("%s not found: %s", resource, id))
}

func ManifestNotFound(w http.ResponseWriter, assetID string) {
	NotFound(w, "manifest", assetID)
}

func LicenseUnauthorized(w http.ResponseWriter, reason string) {
	Unauthorized(w, reason)
}

func LicenseBadGateway(w http.ResponseWriter, drmType string) {
	Error(w, http.StatusBadGateway, ErrCodeBadGateway, fmt.Sprintf("upstream %s license server unavailable", drmType))
}

func NotReady(w http.ResponseWriter, reason string) {
	Error(w, http.StatusServiceUnavailable, ErrCodeUnavailable, reason)
}
