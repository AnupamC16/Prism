package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/anupam-chopra/prism/internal/api/controller"
	"github.com/anupam-chopra/prism/internal/config"
)

func chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

func NewRouter(
	manifestCtrl *controller.ManifestController,
	licenseCtrl *controller.LicenseController,
	tokenCtrl *controller.TokenController,
	healthCtrl *controller.HealthController,
	cfg *config.Config,
	logger *slog.Logger,
) http.Handler {
	mux := http.NewServeMux()

	optionsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /health", healthCtrl.Health)
	mux.HandleFunc("GET /ready", healthCtrl.Ready)

	mux.HandleFunc("POST /token", tokenCtrl.Issue)
	mux.Handle("OPTIONS /token", optionsHandler)

	mux.HandleFunc("GET /manifest/hls/{id}", manifestCtrl.GetHLS)
	mux.HandleFunc("GET /manifest/dash/{id}", manifestCtrl.GetDASH)

	mux.HandleFunc("POST /license/widevine", licenseCtrl.Widevine)
	mux.Handle("OPTIONS /license/widevine", optionsHandler)

	mux.HandleFunc("POST /license/fairplay", licenseCtrl.FairPlay)
	mux.Handle("OPTIONS /license/fairplay", optionsHandler)

	mux.HandleFunc("POST /license/playready", licenseCtrl.PlayReady)
	mux.Handle("OPTIONS /license/playready", optionsHandler)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"success":false,"error":{"code":"NOT_FOUND","message":"route not found"}}`))
	})

	return chain(mux,
		Recovery(logger),
		RequestID(),
		RequestLogger(logger),
		Timeout(25*time.Second),
		CORS(),
	)
}
