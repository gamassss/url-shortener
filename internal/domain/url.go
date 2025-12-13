package domain

import "time"

type URL struct {
	ID          int64      `json:"id"`
	ShortCode   string     `json:"short_code"`
	OriginalURL string     `json:"original_url"`
	Clicks      int64      `json:"clicks"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	IsActive    bool       `json:"is_active"`
}

type CreatedURLRequest struct {
	OriginalURL string `json:"url" binding:"required,url"`
	CustomAlias string `json:"custom_alias,omitempty"`
	ExpiryHours int    `json:"expiry_hours,omitempty"`
}
