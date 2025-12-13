package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gamassss/url-shortener/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type URLRepository struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewURLRepository(db *pgxpool.Pool, redis *redis.Client) *URLRepository {
	return &URLRepository{db: db, redis: redis}
}

func (r *URLRepository) Create(ctx context.Context, url *domain.URL) error {
	query := `
		INSERT INTO urls (short_code, original_url, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRow(ctx, query, url.ShortCode, url.OriginalURL, url.ExpiresAt).Scan(&url.ID, &url.CreatedAt, &url.UpdatedAt)
}

func (r *URLRepository) GetByShortCode(ctx context.Context, shortCode string) (*domain.URL, error) {
	cacheKey := fmt.Sprintf("url:%s", shortCode)

	if cachedData, err := r.redis.Get(ctx, cacheKey).Result(); err == nil {
		var url domain.URL
		if err := json.Unmarshal([]byte(cachedData), &url); err == nil {
			return &url, nil
		}
	}

	var url domain.URL

	query := `
		SELECT id, short_code, original_url, clicks, created_at, updated_at, expires_at, is_active FROM urls
		WHERE short_code = $1	
	`

	row := r.db.QueryRow(ctx, query, shortCode)

	err := row.Scan(
		&url.ID,
		&url.ShortCode,
		&url.OriginalURL,
		&url.Clicks,
		&url.CreatedAt,
		&url.UpdatedAt,
		&url.ExpiresAt,
		&url.IsActive,
	)

	if err != nil {
		return nil, err
	}

	urlJSON, _ := json.Marshal(url)
	ttl := 24 * time.Hour
	if url.ExpiresAt != nil {
		ttl = time.Until(*url.ExpiresAt)
	}

	go func() {
		r.redis.Set(context.Background(), cacheKey, urlJSON, ttl)
	}()

	return &url, nil
}
