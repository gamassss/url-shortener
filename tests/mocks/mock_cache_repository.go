package mocks

import (
	"context"
	"time"

	"github.com/gamassss/url-shortener/internal/domain"
	"github.com/stretchr/testify/mock"
)

type MockCacheRepository struct {
	mock.Mock
}

func (m *MockCacheRepository) GetURL(ctx context.Context, shortCode string) (*domain.URL, error) {
	args := m.Called(ctx, shortCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.URL), args.Error(1)
}

func (m *MockCacheRepository) SetURL(ctx context.Context, url *domain.URL, ttl time.Duration) error {
	args := m.Called(ctx, url, ttl)
	return args.Error(0)
}
