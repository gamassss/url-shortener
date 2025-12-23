package postgres

import (
	"context"
	"time"

	"github.com/gamassss/url-shortener/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnalyticsRepository struct {
	db *pgxpool.Pool
}

func NewAnalyticsRepository(db *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

func (r *AnalyticsRepository) RecordClick(ctx context.Context, click *domain.ClickRequest) error {
	query := `
		INSERT INTO url_clicks (url_id, user_agent, referer, ip_address, device_type)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(ctx, query,
		click.URLID,
		click.UserAgent,
		click.Referer,
		click.IPAddress,
		click.DeviceType,
	)
	return err
}

func (r *AnalyticsRepository) GetAnalytics(ctx context.Context, urlID int64, days int) (*domain.URLAnalytics, error) {
	analytics := &domain.URLAnalytics{}

	query := `
		SELECT 
			u.short_code,
			u.original_url,
			u.click_count,
			u.created_at,
			MAX(c.clicked_at) as last_clicked_at,
			COUNT(DISTINCT c.ip_address) as unique_ips
		FROM urls u
		LEFT JOIN url_clicks c ON u.id = c.url_id
		WHERE u.id = $1
		GROUP BY u.id, u.short_code, u.original_url, u.click_count, u.created_at
	`

	var lastClickedAt *time.Time
	err := r.db.QueryRow(ctx, query, urlID).Scan(
		&analytics.ShortCode,
		&analytics.OriginalURL,
		&analytics.TotalClicks,
		&analytics.CreatedAt,
		&lastClickedAt,
		&analytics.UniqueIPs,
	)
	if err != nil {
		return nil, err
	}
	analytics.LastClickedAt = lastClickedAt

	clicksByDate, err := r.getClicksByDate(ctx, urlID, days)
	if err != nil {
		return nil, err
	}
	analytics.ClicksByDate = clicksByDate

	topReferrers, err := r.getTopReferrers(ctx, urlID, 5)
	if err != nil {
		return nil, err
	}
	analytics.TopReferrers = topReferrers

	deviceStats, err := r.getDeviceStats(ctx, urlID)
	if err != nil {
		return nil, err
	}
	analytics.DeviceStats = *deviceStats

	return analytics, nil
}

func (r *AnalyticsRepository) getClicksByDate(ctx context.Context, urlID int64, days int) ([]domain.ClicksByDate, error) {
	query := `
		SELECT 
			DATE(clicked_at) as date,
			COUNT(*) as count
		FROM url_clicks
		WHERE url_id = $1 
			AND clicked_at >= NOW() - INTERVAL '1 day' * $2
		GROUP BY DATE(clicked_at)
		ORDER BY date DESC
		LIMIT 30
	`

	rows, err := r.db.Query(ctx, query, urlID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.ClicksByDate
	for rows.Next() {
		var cbd domain.ClicksByDate
		var date time.Time
		if err := rows.Scan(&date, &cbd.Count); err != nil {
			return nil, err
		}
		cbd.Date = date.Format("2006-01-02")
		results = append(results, cbd)
	}

	return results, rows.Err()
}

func (r *AnalyticsRepository) getTopReferrers(ctx context.Context, urlID int64, limit int) ([]domain.ReferrerStats, error) {
	query := `
		SELECT 
			COALESCE(NULLIF(referer, ''), 'Direct') as referer,
			COUNT(*) as count
		FROM url_clicks
		WHERE url_id = $1
		GROUP BY referer
		ORDER BY count DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, urlID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.ReferrerStats
	for rows.Next() {
		var rs domain.ReferrerStats
		if err := rows.Scan(&rs.Referer, &rs.Count); err != nil {
			return nil, err
		}
		results = append(results, rs)
	}

	return results, rows.Err()
}

func (r *AnalyticsRepository) getDeviceStats(ctx context.Context, urlID int64) (*domain.DeviceStats, error) {
	query := `
		SELECT 
			COALESCE(device_type, 'unknown') as device_type,
			COUNT(*) as count
		FROM url_clicks
		WHERE url_id = $1
		GROUP BY device_type
	`

	rows, err := r.db.Query(ctx, query, urlID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := &domain.DeviceStats{}
	for rows.Next() {
		var deviceType string
		var count int64
		if err := rows.Scan(&deviceType, &count); err != nil {
			return nil, err
		}

		switch deviceType {
		case "mobile":
			stats.Mobile = count
		case "desktop":
			stats.Desktop = count
		case "tablet":
			stats.Tablet = count
		case "bot":
			stats.Bot = count
		default:
			stats.Unknown = count
		}
	}

	return stats, rows.Err()
}

func (r *AnalyticsRepository) GetClickHistory(ctx context.Context, urlID int64, page, pageSize int) (*domain.ClickHistory, error) {
	offset := (page - 1) * pageSize

	var total int64
	countQuery := `SELECT COUNT(*) FROM url_clicks WHERE url_id = $1`
	err := r.db.QueryRow(ctx, countQuery, urlID).Scan(&total)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, url_id, clicked_at, user_agent, referer, ip_address, device_type
		FROM url_clicks
		WHERE url_id = $1
		ORDER BY clicked_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, urlID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clicks []domain.URLClick
	for rows.Next() {
		var click domain.URLClick
		err := rows.Scan(
			&click.ID,
			&click.URLID,
			&click.ClickedAt,
			&click.UserAgent,
			&click.Referer,
			&click.IPAddress,
			&click.DeviceType,
		)
		if err != nil {
			return nil, err
		}
		clicks = append(clicks, click)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &domain.ClickHistory{
		Clicks:     clicks,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, rows.Err()
}
