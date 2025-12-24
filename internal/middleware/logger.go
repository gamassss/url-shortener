package middleware

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/gamassss/url-shortener/internal/logger"
	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		requestID := logger.NewRequestID()
		c.Header("X-Request-ID", requestID)

		ctx := logger.WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		log := logger.FromContext(ctx)

		path := c.Request.URL.Path
		if c.Request.URL.RawQuery != "" {
			path = path + "?" + c.Request.URL.RawQuery
		}

		log.Info("HTTP request started",
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("ip", c.ClientIP()),
			slog.String("user_agent", c.Request.UserAgent()),
		)

		c.Next()

		duration := time.Since(start)

		logLevel := slog.LevelInfo
		if c.Writer.Status() >= 500 {
			logLevel = slog.LevelError
		} else if c.Writer.Status() >= 400 {
			logLevel = slog.LevelWarn
		}

		log.Log(c.Request.Context(), logLevel, "HTTP request completed",
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("duration", duration),
			slog.Int("size", c.Writer.Size()),
			slog.String("ip", c.ClientIP()),
		)

		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				log.Error("Request error occurred",
					slog.String("error", err.Error()),
					slog.String("type", strconv.FormatUint(uint64(err.Type), 10)),
				)
			}
		}
	}
}
