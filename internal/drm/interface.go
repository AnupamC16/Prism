package drm

import (
	"context"

	"github.com/anupam-chopra/prism/internal/model"
)

type DRMProvider interface {
	GetLicense(ctx context.Context, challenge []byte, token string) ([]byte, error)
	Name() string
}

type RouterI interface {
	Route(drmType string) (DRMProvider, error)
}

type Router struct {
	widevine  DRMProvider
	fairplay  DRMProvider
	playready DRMProvider
}

func NewRouter(widevine, fairplay, playready DRMProvider) *Router {
	if widevine == nil {
		panic("drm router: widevine is nil")
	}
	if fairplay == nil {
		panic("drm router: fairplay is nil")
	}
	if playready == nil {
		panic("drm router: playready is nil")
	}
	return &Router{
		widevine:  widevine,
		fairplay:  fairplay,
		playready: playready,
	}
}

func (r *Router) Route(drmType string) (DRMProvider, error) {
	switch drmType {
	case "widevine":
		return r.widevine, nil
	case "fairplay":
		return r.fairplay, nil
	case "playready":
		return r.playready, nil
	default:
		return nil, model.NewDRMError(drmType, "unsupported DRM type")
	}
}
