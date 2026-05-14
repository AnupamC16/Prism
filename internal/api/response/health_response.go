package response

import (
	"net/http"
	"time"
)

type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	UptimeSec int64  `json:"uptime_sec"`
}

func HealthOK(w http.ResponseWriter, version string, startTime time.Time) {
	Success(w, http.StatusOK, HealthResponse{
		Status:    "ok",
		Version:   version,
		UptimeSec: int64(time.Since(startTime).Seconds()),
	})
}

type ReadyResponse struct {
	Status string `json:"status"`
	Cache  string `json:"cache"`
}

func ReadyOK(w http.ResponseWriter) {
	Success(w, http.StatusOK, ReadyResponse{
		Status: "ready",
		Cache:  "ok",
	})
}
