package handler

import (
	"context"
	"net/http"

	"github.com/gamassss/url-shortener/internal/domain"
	"github.com/gamassss/url-shortener/pkg/response"
	"github.com/gamassss/url-shortener/pkg/validator"
	"github.com/gin-gonic/gin"
)

type ShortenerService interface {
	ShortenURL(ctx context.Context, req *domain.CreatedURLRequest) (*domain.URL, error)
	GetOriginalURL(ctx context.Context, shortCode string) (*domain.URL, bool, error)
}

type ShortenerHandler struct {
	service ShortenerService
}

func NewShortenerHandler(service ShortenerService) *ShortenerHandler {
	return &ShortenerHandler{service: service}
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

	response.Created(c, "URL shortened successfully", gin.H{
		"short_url":    "http://localhost:8080/" + url.ShortCode,
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

	if cacheHit {
		c.Header("X-Cache-Hit", "true")
	} else {
		c.Header("X-Cache-Hit", "false")
	}

	c.Redirect(http.StatusMovedPermanently, url.OriginalURL)
}
