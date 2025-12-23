package domain

import "time"

type URLClick struct {
	ID          int64     `json:"id"`
	URLID       int64     `json:"url_id"`
	ClickedAt   time.Time `json:"clicked_at"`
	UserAgent   string    `json:"user_agent"`
	Referer     string    `json:"referer"`
	IPAddress   string    `json:"ip_address"`
	CountryCode string    `json:"country_code,omitempty"`
	DeviceType  string    `json:"device_type"`
}

type ClickRequest struct {
	URLID      int64
	UserAgent  string
	Referer    string
	IPAddress  string
	DeviceType string
}

type URLAnalytics struct {
	ShortCode     string          `json:"short_code"`
	OriginalURL   string          `json:"original_url"`
	TotalClicks   int64           `json:"total_clicks"`
	UniqueIPs     int64           `json:"unique_ips"`
	LastClickedAt *time.Time      `json:"last_clicked_at"`
	CreatedAt     time.Time       `json:"created_at"`
	ClicksByDate  []ClicksByDate  `json:"clicks_by_date"`
	TopReferrers  []ReferrerStats `json:"top_referrers"`
	DeviceStats   DeviceStats     `json:"device_stats"`
}

type ClicksByDate struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type ReferrerStats struct {
	Referer string `json:"referer"`
	Count   int64  `json:"count"`
}

type DeviceStats struct {
	Mobile  int64 `json:"mobile"`
	Desktop int64 `json:"desktop"`
	Tablet  int64 `json:"tablet"`
	Bot     int64 `json:"bot"`
	Unknown int64 `json:"unknown"`
}

type ClickHistory struct {
	Clicks     []URLClick `json:"clicks"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}
