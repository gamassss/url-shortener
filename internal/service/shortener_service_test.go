package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gamassss/url-shortener/internal/domain"
	"github.com/gamassss/url-shortener/tests/mocks"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestShortenURL_Success_GeneratedCode(t *testing.T) {
	mockURLRepo := new(mocks.MockURLRepository)
	mockCacheRepo := new(mocks.MockCacheRepository)
	service := NewShortenerService(mockURLRepo, mockCacheRepo)
	ctx := context.Background()

	req := &domain.CreatedURLRequest{
		OriginalURL: "https://example.com",
	}

	mockURLRepo.On("Create", ctx, mock.MatchedBy(func(url *domain.URL) bool {
		return url.OriginalURL == "https://example.com" &&
			len(url.ShortCode) == 7 &&
			url.IsActive == true &&
			url.ExpiresAt == nil
	})).Return(nil).Once()

	result, err := service.ShortenURL(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "https://example.com", result.OriginalURL)
	assert.Len(t, result.ShortCode, 7)
	assert.True(t, result.IsActive)
	mockURLRepo.AssertExpectations(t)
	mockCacheRepo.AssertExpectations(t)
}

func TestShortenURL_Success_CustomAlias(t *testing.T) {
	mockURLRepo := new(mocks.MockURLRepository)
	mockCacheRepo := new(mocks.MockCacheRepository)
	service := NewShortenerService(mockURLRepo, mockCacheRepo)
	ctx := context.Background()

	req := &domain.CreatedURLRequest{
		OriginalURL: "https://example.com",
		CustomAlias: "mylink",
	}

	mockURLRepo.On("Create", ctx, mock.MatchedBy(func(url *domain.URL) bool {
		return url.ShortCode == "mylink" &&
			url.OriginalURL == "https://example.com"
	})).Return(nil).Once()

	result, err := service.ShortenURL(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, "mylink", result.ShortCode)
	mockURLRepo.AssertExpectations(t)
	mockCacheRepo.AssertExpectations(t)
}

func TestShortenURL_Success_WithExpiry(t *testing.T) {
	mockURLRepo := new(mocks.MockURLRepository)
	mockCacheRepo := new(mocks.MockCacheRepository)
	service := NewShortenerService(mockURLRepo, mockCacheRepo)
	ctx := context.Background()

	req := &domain.CreatedURLRequest{
		OriginalURL: "https://example.com",
		ExpiryHours: 24,
	}

	mockURLRepo.On("Create", ctx, mock.MatchedBy(func(url *domain.URL) bool {
		if url.ExpiresAt == nil {
			return false
		}

		expectedExpiry := time.Now().Add(24 * time.Hour)
		diff := url.ExpiresAt.Sub(expectedExpiry)
		return diff < time.Minute && diff > -time.Minute
	})).Return(nil).Once()

	result, err := service.ShortenURL(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result.ExpiresAt)
	mockURLRepo.AssertExpectations(t)
	mockCacheRepo.AssertExpectations(t)
}

func TestShortenURL_Retry_SuccessAfterCollision(t *testing.T) {
	mockURLRepo := new(mocks.MockURLRepository)
	mockCacheRepo := new(mocks.MockCacheRepository)
	service := NewShortenerService(mockURLRepo, mockCacheRepo)
	ctx := context.Background()

	req := &domain.CreatedURLRequest{
		OriginalURL: "https://example.com",
	}

	pgErr := &pgconn.PgError{
		Code:           "23505",
		ConstraintName: "urls_short_code_key",
	}

	mockURLRepo.On("Create", ctx, mock.AnythingOfType("*domain.URL")).
		Return(pgErr).Once()

	mockURLRepo.On("Create", ctx, mock.AnythingOfType("*domain.URL")).
		Return(nil).Once()

	result, err := service.ShortenURL(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	mockURLRepo.AssertNumberOfCalls(t, "Create", 2)
	mockCacheRepo.AssertExpectations(t)
}

func TestShortenURL_Retry_FailAfterMaxRetries(t *testing.T) {
	mockURLRepo := new(mocks.MockURLRepository)
	mockCacheRepo := new(mocks.MockCacheRepository)
	service := NewShortenerService(mockURLRepo, mockCacheRepo)
	ctx := context.Background()

	req := &domain.CreatedURLRequest{
		OriginalURL: "https://example.com",
	}

	pgErr := &pgconn.PgError{
		Code:           "23505",
		ConstraintName: "urls_short_code_key",
	}

	mockURLRepo.On("Create", ctx, mock.AnythingOfType("*domain.URL")).
		Return(pgErr).Times(3)

	result, err := service.ShortenURL(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to generate short code after 3 retries")
	mockURLRepo.AssertNumberOfCalls(t, "Create", 3)
	mockCacheRepo.AssertExpectations(t)
}

func TestShortenURL_CustomAlias_DuplicateError(t *testing.T) {
	mockURLRepo := new(mocks.MockURLRepository)
	mockCacheRepo := new(mocks.MockCacheRepository)
	service := NewShortenerService(mockURLRepo, mockCacheRepo)
	ctx := context.Background()

	req := &domain.CreatedURLRequest{
		OriginalURL: "https://example.com",
		CustomAlias: "existing",
	}

	pgErr := &pgconn.PgError{
		Code:           "23505",
		ConstraintName: "urls_short_code_key",
	}

	mockURLRepo.On("Create", ctx, mock.MatchedBy(func(url *domain.URL) bool {
		return url.ShortCode == "existing"
	})).Return(pgErr).Once()

	result, err := service.ShortenURL(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create short url")

	mockURLRepo.AssertNumberOfCalls(t, "Create", 1)
	mockCacheRepo.AssertExpectations(t)
}

func TestGetOriginalURL_Success_FromCache(t *testing.T) {
	mockURLRepo := new(mocks.MockURLRepository)
	mockCacheRepo := new(mocks.MockCacheRepository)
	service := NewShortenerService(mockURLRepo, mockCacheRepo)
	ctx := context.Background()

	cachedURL := &domain.URL{
		ID:          1,
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		ClickCount:  10,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockCacheRepo.On("GetURL", ctx, "abc123").
		Return(cachedURL, nil).Once()

	result, err := service.GetOriginalURL(ctx, "abc123")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, cachedURL.OriginalURL, result.OriginalURL)
	assert.Equal(t, cachedURL.ShortCode, result.ShortCode)

	mockCacheRepo.AssertExpectations(t)
	mockURLRepo.AssertNotCalled(t, "GetByShortCode")
}

func TestGetOriginalURL_Success_FromDB_CacheMiss(t *testing.T) {
	mockURLRepo := new(mocks.MockURLRepository)
	mockCacheRepo := new(mocks.MockCacheRepository)
	service := NewShortenerService(mockURLRepo, mockCacheRepo)
	ctx := context.Background()

	expectedURL := &domain.URL{
		ID:          1,
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		ClickCount:  10,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockCacheRepo.On("GetURL", ctx, "abc123").
		Return(nil, errors.New("cache miss")).Once()

	mockURLRepo.On("GetByShortCode", ctx, "abc123").
		Return(expectedURL, nil).Once()

	mockCacheRepo.On("SetURL", mock.Anything, expectedURL, mock.AnythingOfType("time.Duration")).
		Return(nil).Maybe()

	result, err := service.GetOriginalURL(ctx, "abc123")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedURL.OriginalURL, result.OriginalURL)
	assert.Equal(t, expectedURL.ShortCode, result.ShortCode)

	mockCacheRepo.AssertCalled(t, "GetURL", ctx, "abc123")
	mockURLRepo.AssertExpectations(t)
}

func TestGetOriginalURL_NotFound(t *testing.T) {
	mockURLRepo := new(mocks.MockURLRepository)
	mockCacheRepo := new(mocks.MockCacheRepository)
	service := NewShortenerService(mockURLRepo, mockCacheRepo)
	ctx := context.Background()

	mockCacheRepo.On("GetURL", ctx, "notfound").
		Return(nil, errors.New("cache miss")).Once()

	mockURLRepo.On("GetByShortCode", ctx, "notfound").
		Return(nil, pgx.ErrNoRows).Once()

	result, err := service.GetOriginalURL(ctx, "notfound")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "URL not found")

	mockCacheRepo.AssertExpectations(t)
	mockURLRepo.AssertExpectations(t)
}

func TestGetOriginalURL_DatabaseError(t *testing.T) {
	mockURLRepo := new(mocks.MockURLRepository)
	mockCacheRepo := new(mocks.MockCacheRepository)
	service := NewShortenerService(mockURLRepo, mockCacheRepo)
	ctx := context.Background()

	dbErr := errors.New("connection timeout")

	mockCacheRepo.On("GetURL", ctx, "abc123").
		Return(nil, errors.New("cache miss")).Once()

	mockURLRepo.On("GetByShortCode", ctx, "abc123").
		Return(nil, dbErr).Once()

	result, err := service.GetOriginalURL(ctx, "abc123")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get original url")

	mockCacheRepo.AssertExpectations(t)
	mockURLRepo.AssertExpectations(t)
}

func TestShortenURL_DatabaseError(t *testing.T) {
	mockURLRepo := new(mocks.MockURLRepository)
	mockCacheRepo := new(mocks.MockCacheRepository)
	service := NewShortenerService(mockURLRepo, mockCacheRepo)
	ctx := context.Background()

	req := &domain.CreatedURLRequest{
		OriginalURL: "https://example.com",
	}

	dbErr := fmt.Errorf("database connection failed")
	mockURLRepo.On("Create", ctx, mock.AnythingOfType("*domain.URL")).
		Return(dbErr).Once()

	result, err := service.ShortenURL(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create short url")

	mockURLRepo.AssertNumberOfCalls(t, "Create", 1)
	mockCacheRepo.AssertExpectations(t)
}

func TestGetOriginalURL_CacheError_FallbackToDB(t *testing.T) {
	mockURLRepo := new(mocks.MockURLRepository)
	mockCacheRepo := new(mocks.MockCacheRepository)
	service := NewShortenerService(mockURLRepo, mockCacheRepo)
	ctx := context.Background()

	expectedURL := &domain.URL{
		ID:          1,
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		IsActive:    true,
	}

	mockCacheRepo.On("GetURL", ctx, "abc123").
		Return(nil, errors.New("redis connection error")).Once()

	mockURLRepo.On("GetByShortCode", ctx, "abc123").
		Return(expectedURL, nil).Once()

	mockCacheRepo.On("SetURL", mock.Anything, expectedURL, mock.AnythingOfType("time.Duration")).
		Return(nil).Maybe()

	result, err := service.GetOriginalURL(ctx, "abc123")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedURL.OriginalURL, result.OriginalURL)

	mockCacheRepo.AssertCalled(t, "GetURL", ctx, "abc123")
	mockURLRepo.AssertExpectations(t)
}

func TestGetOriginalURL_WithExpiry_CorrectTTL(t *testing.T) {
	mockURLRepo := new(mocks.MockURLRepository)
	mockCacheRepo := new(mocks.MockCacheRepository)
	service := NewShortenerService(mockURLRepo, mockCacheRepo)
	ctx := context.Background()

	expiresAt := time.Now().Add(2 * time.Hour)
	expectedURL := &domain.URL{
		ID:          1,
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		ExpiresAt:   &expiresAt,
		IsActive:    true,
	}

	mockCacheRepo.On("GetURL", ctx, "abc123").
		Return(nil, errors.New("cache miss")).Once()

	mockURLRepo.On("GetByShortCode", ctx, "abc123").
		Return(expectedURL, nil).Once()

	mockCacheRepo.On("SetURL", mock.Anything, expectedURL, mock.MatchedBy(func(ttl time.Duration) bool {
		expectedTTL := time.Until(expiresAt)
		diff := ttl - expectedTTL
		return diff < time.Minute && diff > -time.Minute
	})).Return(nil).Maybe()

	result, err := service.GetOriginalURL(ctx, "abc123")

	assert.NoError(t, err)
	assert.NotNil(t, result)

	time.Sleep(100 * time.Millisecond)

	mockCacheRepo.AssertExpectations(t)
	mockURLRepo.AssertExpectations(t)
}
