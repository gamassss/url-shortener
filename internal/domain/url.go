package domain

import "time"

type URL struct {
	ID          int64      `json:"id"`
	ShortCode   string     `json:"short_code"`
	OriginalURL string     `json:"originalURL"`
	Clicks      string     `json:"clicks"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	ExpiresAt   *time.Time `json:"expiresAt"`
	IsActive    bool       `json:"isActive"`
}

type CreatedURLRequest struct {
	OriginalURL string `json:"url" binding:"required, url"`
	CustomAlias string `json:"custom_alias,omitempty"`
	ExpiryHours int    `json:"expiry_hours,omitempty"`
}
