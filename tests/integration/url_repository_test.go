//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gamassss/url-shortener/internal/domain"
	"github.com/gamassss/url-shortener/internal/repository/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	testpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDatabase(t *testing.T) (*pgxpool.Pool, func()) {
	ctx := context.Background()

	pgContainer, err := testpostgres.Run(ctx,
		"postgres:16-alpine",
		testpostgres.WithDatabase("testdb"),
		testpostgres.WithUsername("testuser"),
		testpostgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	dbPool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	err = applyMigration(ctx, dbPool)
	require.NoError(t, err)

	cleanup := func() {
		dbPool.Close()
		pgContainer.Terminate(ctx)
	}

	return dbPool, cleanup
}

func applyMigration(ctx context.Context, db *pgxpool.Pool) error {
	migrationPath := filepath.Join("..", "..", "migrations", "0001_create_urls_table.up.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		return err
	}

	_, err = db.Exec(ctx, string(migrationSQL))
	return err
}

func TestURLRepository_Create_Success(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := postgres.NewURLRepository(db)
	ctx := context.Background()

	url := &domain.URL{
		ShortCode:   "abc1234",
		OriginalURL: "https://example.com",
		IsActive:    true,
	}

	err := repo.Create(ctx, url)

	assert.NoError(t, err)
	assert.NotZero(t, url.ID, "ID should be auto-generated")
	assert.NotZero(t, url.CreatedAt, "CreatedAt should be set")
	assert.NotZero(t, url.UpdatedAt, "UpdatedAt should be set")
}

func TestURLRepository_Create_WithExpiry(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := postgres.NewURLRepository(db)
	ctx := context.Background()

	expiresAt := time.Now().Add(24 * time.Hour)
	url := &domain.URL{
		ShortCode:   "exp1234",
		OriginalURL: "https://example.com",
		ExpiresAt:   &expiresAt,
		IsActive:    true,
	}

	err := repo.Create(ctx, url)

	assert.NoError(t, err)
	assert.NotNil(t, url.ExpiresAt)
}

func TestURLRepository_Create_DuplicateShortCode(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := postgres.NewURLRepository(db)
	ctx := context.Background()

	url1 := &domain.URL{
		ShortCode:   "duplicate",
		OriginalURL: "https://example1.com",
		IsActive:    true,
	}
	err := repo.Create(ctx, url1)
	require.NoError(t, err)

	url2 := &domain.URL{
		ShortCode:   "duplicate",
		OriginalURL: "https://example2.com",
		IsActive:    true,
	}
	err = repo.Create(ctx, url2)

	assert.Error(t, err, "Should return error for duplicate short code")
	assert.Contains(t, err.Error(), "duplicate key")
}

func TestURLRepository_GetByShortCode_Success(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := postgres.NewURLRepository(db)
	ctx := context.Background()

	url := &domain.URL{
		ShortCode:   "fetch123",
		OriginalURL: "https://example.com",
		IsActive:    true,
	}
	err := repo.Create(ctx, url)
	require.NoError(t, err)

	result, err := repo.GetByShortCode(ctx, "fetch123")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "fetch123", result.ShortCode)
	assert.Equal(t, "https://example.com", result.OriginalURL)
	assert.True(t, result.IsActive)
	assert.Equal(t, int64(0), result.Clicks)
}

func TestURLRepository_GetByShortCode_NotFound(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := postgres.NewURLRepository(db)
	ctx := context.Background()

	result, err := repo.GetByShortCode(ctx, "notfound")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestURLRepository_ConcurrentCreation(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := postgres.NewURLRepository(db)
	ctx := context.Background()

	urls := []string{"url1", "url2", "url3", "url4", "url5"}
	errChan := make(chan error, len(urls))

	for i, shortCode := range urls {
		go func(code string, index int) {
			url := &domain.URL{
				ShortCode:   code,
				OriginalURL: "https://example.com/" + code,
				IsActive:    true,
			}
			errChan <- repo.Create(ctx, url)
		}(shortCode, i)
	}

	for range urls {
		err := <-errChan
		assert.NoError(t, err)
	}

	for _, shortCode := range urls {
		result, err := repo.GetByShortCode(ctx, shortCode)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestURLRepository_ExpiredURL(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := postgres.NewURLRepository(db)
	ctx := context.Background()

	expiresAt := time.Now().Add(-24 * time.Hour)
	url := &domain.URL{
		ShortCode:   "expired1",
		OriginalURL: "https://example.com",
		ExpiresAt:   &expiresAt,
		IsActive:    true,
	}
	err := repo.Create(ctx, url)
	require.NoError(t, err)

	result, err := repo.GetByShortCode(ctx, "expired1")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.ExpiresAt)

	now := time.Now()
	assert.True(t, result.ExpiresAt.Before(now),
		"ExpiresAt (%v) should be before now (%v)", result.ExpiresAt, now)
}

func TestURLRepository_BulkOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping bulk operations test in short mode")
	}

	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := postgres.NewURLRepository(db)
	ctx := context.Background()

	count := 100
	start := time.Now()

	for i := 0; i < count; i++ {
		shortCode := fmt.Sprintf("bulk%03d", i)

		url := &domain.URL{
			ShortCode:   shortCode,
			OriginalURL: fmt.Sprintf("https://example.com/bulk%d", i),
			IsActive:    true,
		}
		err := repo.Create(ctx, url)
		require.NoError(t, err, "Failed to create URL with short code: %s", shortCode)
	}

	duration := time.Since(start)
	t.Logf("Created %d URLs in %v (avg: %v per URL)", count, duration, duration/time.Duration(count))

	first, err := repo.GetByShortCode(ctx, "bulk000")
	assert.NoError(t, err)
	assert.NotNil(t, first)
	assert.Equal(t, "https://example.com/bulk0", first.OriginalURL)

	last, err := repo.GetByShortCode(ctx, fmt.Sprintf("bulk%03d", count-1))
	assert.NoError(t, err)
	assert.NotNil(t, last)
	assert.Equal(t, fmt.Sprintf("https://example.com/bulk%d", count-1), last.OriginalURL)
}

func TestURLRepository_MultipleReads(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	repo := postgres.NewURLRepository(db)
	ctx := context.Background()

	url := &domain.URL{
		ShortCode:   "multiread",
		OriginalURL: "https://example.com",
		IsActive:    true,
	}
	err := repo.Create(ctx, url)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		result, err := repo.GetByShortCode(ctx, "multiread")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "https://example.com", result.OriginalURL)
	}
}
