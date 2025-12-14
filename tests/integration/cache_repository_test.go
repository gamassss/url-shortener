//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gamassss/url-shortener/internal/domain"
	redisrepo "github.com/gamassss/url-shortener/internal/repository/redis"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cleanup := func() {
		client.Close()
		mr.Close()
	}

	return client, cleanup
}

func TestCacheRepository_SetAndGetURL(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := redisrepo.NewURLCache(redisClient)
	ctx := context.Background()

	url := &domain.URL{
		ID:          1,
		ShortCode:   "test123",
		OriginalURL: "https://example.com",
		Clicks:      10,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := repo.SetURL(ctx, url, 10*time.Minute)
	require.NoError(t, err)

	result, err := repo.GetURL(ctx, "test123")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, url.ShortCode, result.ShortCode)
	assert.Equal(t, url.OriginalURL, result.OriginalURL)
	assert.Equal(t, url.Clicks, result.Clicks)
	assert.Equal(t, url.IsActive, result.IsActive)
}

func TestCacheRepository_GetURL_NotFound(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := redisrepo.NewURLCache(redisClient)
	ctx := context.Background()

	result, err := repo.GetURL(ctx, "notfound")

	assert.NoError(t, err)
	assert.Nil(t, result, "Should return nil for non-existent key")
}

func TestCacheRepository_SetURL_WithExpiry(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := redisrepo.NewURLCache(redisClient)
	ctx := context.Background()

	expiresAt := time.Now().Add(1 * time.Hour)
	url := &domain.URL{
		ShortCode:   "expiry123",
		OriginalURL: "https://example.com",
		ExpiresAt:   &expiresAt,
		IsActive:    true,
	}

	ttl := 5 * time.Second
	err := repo.SetURL(ctx, url, ttl)
	require.NoError(t, err)

	result, err := repo.GetURL(ctx, "expiry123")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, url.ExpiresAt.Unix(), result.ExpiresAt.Unix())
}

func TestCacheRepository_UpdateURL(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := redisrepo.NewURLCache(redisClient)
	ctx := context.Background()

	url := &domain.URL{
		ShortCode:   "update123",
		OriginalURL: "https://example.com",
		Clicks:      5,
		IsActive:    true,
	}

	err := repo.SetURL(ctx, url, 10*time.Minute)
	require.NoError(t, err)

	url.Clicks = 10
	err = repo.SetURL(ctx, url, 10*time.Minute)
	require.NoError(t, err)

	result, err := repo.GetURL(ctx, "update123")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(10), result.Clicks)
}

func TestCacheRepository_MultipleURLs(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := redisrepo.NewURLCache(redisClient)
	ctx := context.Background()

	urls := []*domain.URL{
		{ShortCode: "url1", OriginalURL: "https://example1.com", IsActive: true},
		{ShortCode: "url2", OriginalURL: "https://example2.com", IsActive: true},
		{ShortCode: "url3", OriginalURL: "https://example3.com", IsActive: true},
	}

	for _, url := range urls {
		err := repo.SetURL(ctx, url, 10*time.Minute)
		require.NoError(t, err)
	}

	for _, url := range urls {
		result, err := repo.GetURL(ctx, url.ShortCode)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, url.OriginalURL, result.OriginalURL)
	}
}

func TestCacheRepository_ConcurrentAccess(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := redisrepo.NewURLCache(redisClient)
	ctx := context.Background()

	url := &domain.URL{
		ShortCode:   "concurrent",
		OriginalURL: "https://example.com",
		IsActive:    true,
	}

	err := repo.SetURL(ctx, url, 10*time.Minute)
	require.NoError(t, err)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			result, err := repo.GetURL(ctx, "concurrent")
			assert.NoError(t, err)
			assert.NotNil(t, result)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestCacheRepository_InvalidJSON(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()

	err := redisClient.Set(ctx, "url:invalid", "not-valid-json", 10*time.Minute).Err()
	require.NoError(t, err)

	repo := redisrepo.NewURLCache(redisClient)

	result, err := repo.GetURL(ctx, "invalid")
	assert.Error(t, err, "Should return error for invalid JSON")
	assert.Nil(t, result)
}

func TestCacheRepository_LargePayload(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	repo := redisrepo.NewURLCache(redisClient)
	ctx := context.Background()

	longURL := "https://example.com/" + string(make([]byte, 1000))
	url := &domain.URL{
		ShortCode:   "large",
		OriginalURL: longURL,
		IsActive:    true,
	}

	err := repo.SetURL(ctx, url, 10*time.Minute)
	require.NoError(t, err)

	result, err := repo.GetURL(ctx, "large")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, longURL, result.OriginalURL)
}
