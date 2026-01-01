package postgres

import (
	"context"

	"github.com/gamassss/url-shortener/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type URLRepository struct {
	db *pgxpool.Pool
}

func NewURLRepository(db *pgxpool.Pool) *URLRepository {
	return &URLRepository{db: db}
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
	var url domain.URL

	query := `
		SELECT id, short_code, original_url, click_count, created_at, updated_at, expires_at, is_active 
		FROM urls
		WHERE short_code = $1 AND is_active = true
		AND (expires_at IS NULL OR expires_at > NOW())
	`

	row := r.db.QueryRow(ctx, query, shortCode)

	err := row.Scan(
		&url.ID,
		&url.ShortCode,
		&url.OriginalURL,
		&url.ClickCount,
		&url.CreatedAt,
		&url.UpdatedAt,
		&url.ExpiresAt,
		&url.IsActive,
	)

	if err != nil {
		return nil, err
	}

	return &url, nil
}
