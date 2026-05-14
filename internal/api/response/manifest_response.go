package response

import (
	"net/http"
)

func HLSManifest(w http.ResponseWriter, data []byte) {
	w.Header().Set("Cache-Control", "public, max-age=30")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	Raw(w, http.StatusOK, "application/vnd.apple.mpegurl", data)
}

func DASHManifest(w http.ResponseWriter, data []byte) {
	w.Header().Set("Cache-Control", "public, max-age=30")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	Raw(w, http.StatusOK, "application/dash+xml", data)
}
