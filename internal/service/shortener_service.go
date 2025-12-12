package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gamassss/url-shortener/internal/domain"
	"github.com/gamassss/url-shortener/pkg/generator"
	"github.com/jackc/pgx/v5/pgconn"
)

type URLRepository interface {
	Create(ctx context.Context, url *domain.URL) error
	GetByShortCode(ctx context.Context, shortCode string) (*domain.URL, error)
}

type ShortenerService struct {
	urlRepo URLRepository
	// TODO: add redis for opt later
}

func NewShortenerService(urlRepo URLRepository) *ShortenerService {
	return &ShortenerService{urlRepo: urlRepo}
}

func (s *ShortenerService) ShortenURL(ctx context.Context, req *domain.CreatedURLRequest) (*domain.URL, error) {
	var err error
	shortCode := req.CustomAlias
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		if shortCode == "" {
			shortCode, err = generator.GenerateShortCode()
			if err != nil {
				return nil, err
			}
		}

		url := &domain.URL{
			OriginalURL: req.OriginalURL,
			ShortCode:   shortCode,
			IsActive:    true,
		}

		if req.ExpiryHours > 0 {
			expires := time.Now().Add(time.Duration(req.ExpiryHours) * time.Hour)
			url.ExpiresAt = &expires
		}

		err = s.urlRepo.Create(ctx, url)
		if err == nil {
			return url, nil
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if req.CustomAlias == "" && strings.Contains(pgErr.ConstraintName, "short_code") {
				shortCode = ""
				continue
			}
		}

		return nil, fmt.Errorf("failed to create short url: %w", err)
	}

	return nil, fmt.Errorf("failed to generate short code after %d retries: %w", maxRetries, err)
}
