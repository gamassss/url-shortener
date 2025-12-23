package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

type HealthResponse struct {
	Status   string           `json:"status"`
	Checks   map[string]Check `json:"checks"`
	Metadata Metadata         `json:"metadata"`
}

type Check struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type Metadata struct {
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}

func NewHealthHandler(db *pgxpool.Pool, redis *redis.Client) *HealthHandler {
	return &HealthHandler{
		db:    db,
		redis: redis,
	}
}

func (h *HealthHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (h *HealthHandler) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checks := make(map[string]Check)
	allHealthy := true

	dbCheck := h.checkDatabase(ctx)
	checks["database"] = dbCheck
	if dbCheck.Status != "up" {
		allHealthy = false
	}

	redisCheck := h.checkRedis(ctx)
	checks["redis"] = redisCheck
	if redisCheck.Status != "up" {
		allHealthy = false
	}

	response := HealthResponse{
		Status: "up",
		Checks: checks,
		Metadata: Metadata{
			Version:   "1.0.0",
			Timestamp: time.Now().Format(time.RFC3339),
		},
	}

	if !allHealthy {
		response.Status = "down"
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *HealthHandler) checkDatabase(ctx context.Context) Check {
	if err := h.db.Ping(ctx); err != nil {
		return Check{
			Status:  "down",
			Message: err.Error(),
		}
	}

	_ = h.db.Stat()
	return Check{
		Status:  "up",
		Message: "connected",
	}
}

func (h *HealthHandler) checkRedis(ctx context.Context) Check {
	if err := h.redis.Ping(ctx).Err(); err != nil {
		return Check{
			Status:  "down",
			Message: err.Error(),
		}
	}

	return Check{
		Status:  "up",
		Message: "connected",
	}
}
