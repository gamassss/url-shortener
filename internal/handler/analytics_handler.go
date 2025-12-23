package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gamassss/url-shortener/internal/domain"
	"github.com/gamassss/url-shortener/pkg/response"
	"github.com/gin-gonic/gin"
)

type AnalyticsService interface {
	GetAnalytics(ctx context.Context, shortCode string, days int) (*domain.URLAnalytics, error)
	GetClickHistory(ctx context.Context, shortCode string, page, pageSize int) (*domain.ClickHistory, error)
}

type AnalyticsHandler struct {
	service AnalyticsService
}

func NewAnalyticsHandler(service AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{service: service}
}

func (h *AnalyticsHandler) GetAnalytics(c *gin.Context) {
	shortCode := c.Param("shortCode")
	if shortCode == "" {
		response.BadRequest(c, "Short code is required")
		return
	}

	days := 30
	if daysParam := c.Query("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	analytics, err := h.service.GetAnalytics(c.Request.Context(), shortCode, days)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Analytics retrieved successfully", analytics)
}

func (h *AnalyticsHandler) GetClickHistory(c *gin.Context) {
	shortCode := c.Param("shortCode")
	if shortCode == "" {
		response.BadRequest(c, "Short code is required")
		return
	}

	page := 1
	if pageParam := c.Query("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if sizeParam := c.Query("page_size"); sizeParam != "" {
		if s, err := strconv.Atoi(sizeParam); err == nil && s > 0 && s <= 100 {
			pageSize = s
		}
	}

	history, err := h.service.GetClickHistory(c.Request.Context(), shortCode, page, pageSize)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Click history retrieved successfully", history)
}
