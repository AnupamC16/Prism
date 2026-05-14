package response

import "net/http"

func LicenseBytes(w http.ResponseWriter, drmType string, data []byte) {
	contentType := "application/octet-stream"
	if drmType == "playready" {
		contentType = "text/xml; charset=utf-8"
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	Raw(w, http.StatusOK, contentType, data)
}
