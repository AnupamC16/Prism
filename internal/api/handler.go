package api

import (
	"log/slog"

	"github.com/anupam-chopra/prism/internal/api/controller"
)

type Handler struct {
	Manifest *controller.ManifestController
	License  *controller.LicenseController
	Token    *controller.TokenController
	Health   *controller.HealthController
	Logger   *slog.Logger
}

func NewHandler(
	manifest *controller.ManifestController,
	license *controller.LicenseController,
	token *controller.TokenController,
	health *controller.HealthController,
	logger *slog.Logger,
) *Handler {
	if manifest == nil {
		panic("api handler requires manifest controller")
	}
	if license == nil {
		panic("api handler requires license controller")
	}
	if token == nil {
		panic("api handler requires token controller")
	}
	if health == nil {
		panic("api handler requires health controller")
	}

	return &Handler{
		Manifest: manifest,
		License:  license,
		Token:    token,
		Health:   health,
		Logger:   logger,
	}
}
