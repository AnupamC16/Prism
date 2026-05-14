package mock

import (
	"context"
	"sync"
)

type MockDRMProvider struct {
	NameValue       string
	GetLicenseFunc  func(ctx context.Context, challenge []byte, token string) ([]byte, error)
	GetLicenseCalls []struct {
		Challenge []byte
		Token     string
	}
	mu sync.Mutex
}

func (m *MockDRMProvider) Name() string {
	if m.NameValue != "" {
		return m.NameValue
	}
	return "mock"
}

func (m *MockDRMProvider) GetLicense(ctx context.Context, challenge []byte, token string) ([]byte, error) {
	m.mu.Lock()
	m.GetLicenseCalls = append(m.GetLicenseCalls, struct {
		Challenge []byte
		Token     string
	}{Challenge: challenge, Token: token})
	fn := m.GetLicenseFunc
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, challenge, token)
	}
	return []byte("mock-license"), nil
}

func (m *MockDRMProvider) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GetLicenseCalls = nil
}
