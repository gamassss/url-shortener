package handler

import (
	"net/http"

	"github.com/gamassss/url-shortener/internal/domain"
	"github.com/gamassss/url-shortener/internal/service"
	"github.com/gin-gonic/gin"
)

type ShortenerHandler struct {
	service *service.ShortenerService
}

func NewShortenerHandler(service *service.ShortenerService) *ShortenerHandler {
	return &ShortenerHandler{service: service}
}

func (h *ShortenerHandler) ShortenURL(c *gin.Context) {
	var req domain.CreatedURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	url, err := h.service.ShortenURL(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"short_url":    "http://localhost:8080/" + url.ShortCode,
		"short_code":   url.ShortCode,
		"original_url": url.OriginalURL,
		"expires_at":   url.ExpiresAt,
	})
}

func (h *ShortenerHandler) Redirect(c *gin.Context) {
	shortCode := c.Param("shortCode")

	url, err := h.service.GetOriginalURL(c.Request.Context(), shortCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.Redirect(http.StatusMovedPermanently, url.OriginalURL)
}
