package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gamassss/url-shortener/internal/domain"
	"github.com/gamassss/url-shortener/pkg/generator"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type URLRepository interface {
	Create(ctx context.Context, url *domain.URL) error
	GetByShortCode(ctx context.Context, shortCode string) (*domain.URL, error)
}

type CacheRepository interface {
	GetURL(ctx context.Context, shortCode string) (*domain.URL, error)
	SetURL(ctx context.Context, url *domain.URL, ttl time.Duration) error
}

type ShortenerService struct {
	urlRepo   URLRepository
	cacheRepo CacheRepository
}

func NewShortenerService(urlRepo URLRepository, cacheRepo CacheRepository) *ShortenerService {
	return &ShortenerService{urlRepo: urlRepo, cacheRepo: cacheRepo}
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
			OriginalURL: req.URL,
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

func (s *ShortenerService) GetOriginalURL(ctx context.Context, shortCode string) (*domain.URL, bool, error) {
	url, err := s.cacheRepo.GetURL(ctx, shortCode)
	if err == nil && url != nil {
		return url, true, nil
	}

	url, err = s.urlRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, fmt.Errorf("URL not found")
		}
		return nil, false, fmt.Errorf("failed to get original url: %w", err)
	}

	go func() {
		ttl := 24 * time.Hour
		if url.ExpiresAt != nil {
			ttl = time.Until(*url.ExpiresAt)
		}
		s.cacheRepo.SetURL(context.Background(), url, ttl)
	}()

	return url, false, nil
}
