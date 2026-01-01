package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gamassss/url-shortener/internal/domain"
	"github.com/gamassss/url-shortener/pkg/detector"
	"github.com/gamassss/url-shortener/pkg/response"
	"github.com/gamassss/url-shortener/pkg/validator"
	"github.com/gin-gonic/gin"
)

type ShortenerService interface {
	ShortenURL(ctx context.Context, req *domain.CreatedURLRequest) (*domain.URL, error)
	GetOriginalURL(ctx context.Context, shortCode string) (*domain.URL, bool, error)
	RecordClick(ctx context.Context, click *domain.ClickRequest) error
}

type ShortenerHandler struct {
	service ShortenerService
	baseURL string
}

func NewShortenerHandler(service ShortenerService, baseURL string) *ShortenerHandler {
	return &ShortenerHandler{service: service, baseURL: baseURL}
}

func (h *ShortenerHandler) ShortenURL(c *gin.Context) {
	var req domain.CreatedURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid JSON format")
		return
	}

	if validationErros := validator.Validate(req); len(validationErros) > 0 {
		response.ValidationErrors(c, validationErros)
		return
	}

	if req.CustomAlias != "" && validator.IsReservedKeyword(req.CustomAlias) {
		response.BadRequest(c, "This alias cannot be used")
		return
	}

	url, err := h.service.ShortenURL(c.Request.Context(), &req)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	baseURL := h.baseURL
	if baseURL == "" {
		scheme := "https"
		if c.Request.TLS == nil {
			if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
				scheme = proto
			} else {
				scheme = "http"
			}
		}
		baseURL = fmt.Sprintf("%s://%s", scheme, c.Request.Host)
	}

	response.Created(c, "URL shortened successfully", gin.H{
		"short_url":    h.baseURL + "/" + url.ShortCode,
		"short_code":   url.ShortCode,
		"original_url": url.OriginalURL,
		"expires_at":   url.ExpiresAt,
	})
}

func (h *ShortenerHandler) Redirect(c *gin.Context) {
	shortCode := c.Param("shortCode")

	if shortCode == "" {
		response.BadRequest(c, "Short code is required")
		return
	}

	url, cacheHit, err := h.service.GetOriginalURL(c.Request.Context(), shortCode)
	if err != nil {
		response.NotFound(c, "URL not found")
		return
	}

	go func() {
		userAgent := c.Request.UserAgent()
		referer := c.Request.Referer()
		clientIP := detector.GetClientIP(
			c.Request.RemoteAddr,
			c.Request.Header.Get("X-Forwarded-For"),
			c.Request.Header.Get("X-Real-IP"),
		)
		deviceType := detector.DetectDeviceType(userAgent)

		clickReq := &domain.ClickRequest{
			URLID:      url.ID,
			UserAgent:  userAgent,
			Referer:    referer,
			IPAddress:  clientIP,
			DeviceType: deviceType,
		}

		_ = h.service.RecordClick(context.Background(), clickReq)
	}()

	if cacheHit {
		c.Header("X-Cache-Hit", "true")
	} else {
		c.Header("X-Cache-Hit", "false")
	}

	c.Redirect(http.StatusMovedPermanently, url.OriginalURL)
}
