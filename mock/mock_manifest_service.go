package mock

import (
	"context"
	"sync"

	"github.com/anupam-chopra/prism/internal/model"
)

type MockManifestService struct {
	GetHLSFunc  func(ctx context.Context, req *model.ManifestRequest) ([]byte, error)
	GetDASHFunc func(ctx context.Context, req *model.ManifestRequest) ([]byte, error)
	GetHLSCalls  []*model.ManifestRequest
	GetDASHCalls []*model.ManifestRequest
	mu           sync.Mutex
}

func (m *MockManifestService) GetHLS(ctx context.Context, req *model.ManifestRequest) ([]byte, error) {
	m.mu.Lock()
	m.GetHLSCalls = append(m.GetHLSCalls, req)
	fn := m.GetHLSFunc
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, req)
	}
	return []byte("#EXTM3U\n#EXT-X-VERSION:6\n"), nil
}

func (m *MockManifestService) GetDASH(ctx context.Context, req *model.ManifestRequest) ([]byte, error) {
	m.mu.Lock()
	m.GetDASHCalls = append(m.GetDASHCalls, req)
	fn := m.GetDASHFunc
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, req)
	}
	return []byte(`<?xml version="1.0"?><MPD xmlns="urn:mpeg:dash:schema:mpd:2011"/>`), nil
}

func (m *MockManifestService) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GetHLSCalls = nil
	m.GetDASHCalls = nil
}
