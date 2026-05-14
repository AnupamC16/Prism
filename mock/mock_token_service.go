package mock

import (
	"context"
	"sync"
	"time"

	"github.com/anupam-chopra/prism/internal/model"
)

type MockTokenService struct {
	IssueFunc    func(ctx context.Context, token *model.Token) (*model.Token, error)
	ValidateFunc func(ctx context.Context, tokenStr, assetID string) (*model.TokenClaims, error)
	RevokeFunc   func(ctx context.Context, jti string) error
	IssueCalls    []*model.Token
	ValidateCalls []struct{ TokenStr, AssetID string }
	RevokeCalls   []string
	mu            sync.Mutex
}

func (m *MockTokenService) Issue(ctx context.Context, token *model.Token) (*model.Token, error) {
	m.mu.Lock()
	m.IssueCalls = append(m.IssueCalls, token)
	fn := m.IssueFunc
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, token)
	}
	token.JTI = "mock-jti"
	token.SignedString = "mock.jwt.token"
	token.ExpiresAt = time.Now().UTC().Add(time.Hour)
	return token, nil
}

func (m *MockTokenService) Validate(ctx context.Context, tokenStr, assetID string) (*model.TokenClaims, error) {
	m.mu.Lock()
	m.ValidateCalls = append(m.ValidateCalls, struct{ TokenStr, AssetID string }{tokenStr, assetID})
	fn := m.ValidateFunc
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, tokenStr, assetID)
	}
	return &model.TokenClaims{
		JTI:      "mock-jti",
		AssetID:  assetID,
		ViewerID: "mock-viewer",
		Issuer:   "prism",
	}, nil
}

func (m *MockTokenService) Revoke(ctx context.Context, jti string) error {
	m.mu.Lock()
	m.RevokeCalls = append(m.RevokeCalls, jti)
	fn := m.RevokeFunc
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, jti)
	}
	return nil
}

func (m *MockTokenService) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.IssueCalls = nil
	m.ValidateCalls = nil
	m.RevokeCalls = nil
}
