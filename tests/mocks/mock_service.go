package mocks

import (
	"context"

	"github.com/gamassss/url-shortener/internal/domain"
	"github.com/stretchr/testify/mock"
)

type MockShortenerService struct {
	mock.Mock
}

var _ interface {
	ShortenURL(ctx context.Context, req *domain.CreatedURLRequest) (*domain.URL, error)
	GetOriginalURL(ctx context.Context, shortCode string) (*domain.URL, bool, error)
} = (*MockShortenerService)(nil)

func (m *MockShortenerService) ShortenURL(ctx context.Context, req *domain.CreatedURLRequest) (*domain.URL, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.URL), args.Error(1)
}

func (m *MockShortenerService) GetOriginalURL(ctx context.Context, shortCode string) (*domain.URL, bool, error) {
	args := m.Called(ctx, shortCode)
	if args.Get(0) == nil {
		return nil, false, args.Error(1)
	}
	return args.Get(0).(*domain.URL), false, args.Error(1)
}
