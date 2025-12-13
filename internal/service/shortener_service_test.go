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
	mockRepo := new(mocks.MockURLRepository)
	service := NewShortenerService(mockRepo)
	ctx := context.Background()

	req := &domain.CreatedURLRequest{
		OriginalURL: "https://example.com",
	}

	mockRepo.On("Create", ctx, mock.MatchedBy(func(url *domain.URL) bool {
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
	mockRepo.AssertExpectations(t)
}

func TestShortenURL_Success_CustomAlias(t *testing.T) {
	mockRepo := new(mocks.MockURLRepository)
	service := NewShortenerService(mockRepo)
	ctx := context.Background()

	req := &domain.CreatedURLRequest{
		OriginalURL: "https://example.com",
		CustomAlias: "mylink",
	}

	mockRepo.On("Create", ctx, mock.MatchedBy(func(url *domain.URL) bool {
		return url.ShortCode == "mylink" &&
			url.OriginalURL == "https://example.com"
	})).Return(nil).Once()

	result, err := service.ShortenURL(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, "mylink", result.ShortCode)
	mockRepo.AssertExpectations(t)
}

func TestShortenURL_Success_WithExpiry(t *testing.T) {
	mockRepo := new(mocks.MockURLRepository)
	service := NewShortenerService(mockRepo)
	ctx := context.Background()

	req := &domain.CreatedURLRequest{
		OriginalURL: "https://example.com",
		ExpiryHours: 24,
	}

	mockRepo.On("Create", ctx, mock.MatchedBy(func(url *domain.URL) bool {
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
	mockRepo.AssertExpectations(t)
}

func TestShortenURL_Retry_SuccessAfterCollision(t *testing.T) {
	mockRepo := new(mocks.MockURLRepository)
	service := NewShortenerService(mockRepo)
	ctx := context.Background()

	req := &domain.CreatedURLRequest{
		OriginalURL: "https://example.com",
	}

	pgErr := &pgconn.PgError{
		Code:           "23505",
		ConstraintName: "urls_short_code_key",
	}

	mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.URL")).
		Return(pgErr).Once()

	mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.URL")).
		Return(nil).Once()

	result, err := service.ShortenURL(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	mockRepo.AssertNumberOfCalls(t, "Create", 2)
}

func TestShortenURL_Retry_FailAfterMaxRetries(t *testing.T) {
	mockRepo := new(mocks.MockURLRepository)
	service := NewShortenerService(mockRepo)
	ctx := context.Background()

	req := &domain.CreatedURLRequest{
		OriginalURL: "https://example.com",
	}

	pgErr := &pgconn.PgError{
		Code:           "23505",
		ConstraintName: "urls_short_code_key",
	}

	mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.URL")).
		Return(pgErr).Times(3)

	result, err := service.ShortenURL(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to generate short code after 3 retries")
	mockRepo.AssertNumberOfCalls(t, "Create", 3)
}

func TestShortenURL_CustomAlias_DuplicateError(t *testing.T) {
	mockRepo := new(mocks.MockURLRepository)
	service := NewShortenerService(mockRepo)
	ctx := context.Background()

	req := &domain.CreatedURLRequest{
		OriginalURL: "https://example.com",
		CustomAlias: "existing",
	}

	pgErr := &pgconn.PgError{
		Code:           "23505",
		ConstraintName: "urls_short_code_key",
	}

	mockRepo.On("Create", ctx, mock.MatchedBy(func(url *domain.URL) bool {
		return url.ShortCode == "existing"
	})).Return(pgErr).Once()

	result, err := service.ShortenURL(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create short url")

	mockRepo.AssertNumberOfCalls(t, "Create", 1)
}

func TestGetOriginalURL_Success(t *testing.T) {
	mockRepo := new(mocks.MockURLRepository)
	service := NewShortenerService(mockRepo)
	ctx := context.Background()

	expectedURL := &domain.URL{
		ID:          1,
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		Clicks:      10,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockRepo.On("GetByShortCode", ctx, "abc123").
		Return(expectedURL, nil).Once()

	result, err := service.GetOriginalURL(ctx, "abc123")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedURL.OriginalURL, result.OriginalURL)
	assert.Equal(t, expectedURL.ShortCode, result.ShortCode)
	mockRepo.AssertExpectations(t)
}

func TestGetOriginalURL_NotFound(t *testing.T) {
	mockRepo := new(mocks.MockURLRepository)
	service := NewShortenerService(mockRepo)
	ctx := context.Background()

	mockRepo.On("GetByShortCode", ctx, "notfound").
		Return(nil, pgx.ErrNoRows).Once()

	result, err := service.GetOriginalURL(ctx, "notfound")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "URL not found")
	mockRepo.AssertExpectations(t)
}

func TestGetOriginalURL_DatabaseError(t *testing.T) {
	mockRepo := new(mocks.MockURLRepository)
	service := NewShortenerService(mockRepo)
	ctx := context.Background()

	dbErr := errors.New("connection timeout")
	mockRepo.On("GetByShortCode", ctx, "abc123").
		Return(nil, dbErr).Once()

	result, err := service.GetOriginalURL(ctx, "abc123")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get original url")
	mockRepo.AssertExpectations(t)
}

func TestShortenURL_DatabaseError(t *testing.T) {
	mockRepo := new(mocks.MockURLRepository)
	service := NewShortenerService(mockRepo)
	ctx := context.Background()

	req := &domain.CreatedURLRequest{
		OriginalURL: "https://example.com",
	}

	dbErr := fmt.Errorf("database connection failed")
	mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.URL")).
		Return(dbErr).Once()

	result, err := service.ShortenURL(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create short url")

	mockRepo.AssertNumberOfCalls(t, "Create", 1)
}
