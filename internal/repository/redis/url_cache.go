package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gamassss/url-shortener/internal/domain"
	"github.com/redis/go-redis/v9"
)

type URLCache struct {
	client *redis.Client
}

func NewURLCache(client *redis.Client) *URLCache {
	return &URLCache{client: client}
}

func (r *URLCache) GetURL(ctx context.Context, shortCode string) (*domain.URL, error) {
	key := fmt.Sprintf("url:%s", shortCode)

	data, err := r.client.Get(ctx, key).Result()

	if err == redis.Nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	var url domain.URL
	if err := json.Unmarshal([]byte(data), &url); err != nil {
		return nil, err
	}

	return &url, nil
}

func (r *URLCache) SetURL(ctx context.Context, url *domain.URL, ttl time.Duration) error {
	key := fmt.Sprintf("url:%s", url.ShortCode)

	data, err := json.Marshal(url)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, data, ttl).Err()
}
