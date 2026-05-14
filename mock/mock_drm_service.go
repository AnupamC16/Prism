package mock

import (
	"context"
	"sync"

	"github.com/anupam-chopra/prism/internal/model"
)

type MockDRMService struct {
	GetLicenseFunc  func(ctx context.Context, req *model.LicenseRequest) ([]byte, error)
	GetLicenseCalls []*model.LicenseRequest
	mu              sync.Mutex
}

func (m *MockDRMService) GetLicense(ctx context.Context, req *model.LicenseRequest) ([]byte, error) {
	m.mu.Lock()
	m.GetLicenseCalls = append(m.GetLicenseCalls, req)
	fn := m.GetLicenseFunc
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, req)
	}
	return []byte("mock-license-bytes"), nil
}

func (m *MockDRMService) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GetLicenseCalls = nil
}
