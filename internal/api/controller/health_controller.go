package controller

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/anupam-chopra/prism/internal/api/response"
	"github.com/anupam-chopra/prism/internal/cache"
)

type HealthController struct {
	startTime time.Time
	version   string
	cache     cache.Cache
	logger    *slog.Logger
}

func NewHealthController(cache cache.Cache, version string, logger *slog.Logger) *HealthController {
	if cache == nil {
		panic("cache cannot be nil")
	}
	if logger == nil {
		panic("logger cannot be nil")
	}

	return &HealthController{
		startTime: time.Now(),
		version:   version,
		cache:     cache,
		logger:    logger,
	}
}

func (c *HealthController) Health(w http.ResponseWriter, r *http.Request) {
	response.HealthOK(w, c.version, c.startTime)
}

func (c *HealthController) Ready(w http.ResponseWriter, r *http.Request) {
	if err := c.cache.Ping(r.Context()); err != nil {
		response.NotReady(w, "cache unavailable")
		return
	}
	response.ReadyOK(w)
}
