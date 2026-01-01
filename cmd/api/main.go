package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gamassss/url-shortener/internal/config"
	"github.com/gamassss/url-shortener/internal/handler"
	"github.com/gamassss/url-shortener/internal/logger"
	"github.com/gamassss/url-shortener/internal/middleware"
	"github.com/gamassss/url-shortener/internal/repository/postgres"
	redisRepo "github.com/gamassss/url-shortener/internal/repository/redis"
	"github.com/gamassss/url-shortener/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	loggerConfig := logger.Config{
		Level:      cfg.Log.Level,
		Format:     cfg.Log.Format,
		OutputPath: cfg.Log.OutputPath,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
	}

	if err := logger.Initialize(loggerConfig); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	log := logger.Get()
	log.Info("Starting URL Shortener service",
		"port", cfg.Server.Port,
		"log_level", cfg.Log.Level,
	)

	dbPool, err := setupDatabase(cfg)
	if err != nil {
		log.Error("Failed to setup database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	redisClient, err := setupRedis(cfg)
	if err != nil {
		log.Error("Failed to setup redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	urlRepo := postgres.NewURLRepository(dbPool)
	urlCache := redisRepo.NewURLCache(redisClient)
	analyticsRepo := postgres.NewAnalyticsRepository(dbPool)

	shortenerService := service.NewShortenerService(urlRepo, urlCache, analyticsRepo)

	shortenerHandler := handler.NewShortenerHandler(shortenerService, cfg.Server.BaseURL)
	analyticsHandler := handler.NewAnalyticsHandler(shortenerService)
	healthHandler := handler.NewHealthHandler(dbPool, redisClient)

	router := setupRouter(shortenerHandler, analyticsHandler, healthHandler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Info("Server listening", "address", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	gracefulShutdown(srv, cfg.Server.ShutdownTimeout, dbPool, redisClient, log)
}

func setupDatabase(cfg *config.Config) (*pgxpool.Pool, error) {
	dbConfig := cfg.Database
	poolConfig, err := pgxpool.ParseConfig(dbConfig.URL)
	if err != nil {
		return nil, err
	}

	poolConfig.MaxConns = int32(dbConfig.MaxConns)
	poolConfig.MinConns = int32(dbConfig.MinConns)
	poolConfig.MaxConnLifetime = dbConfig.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = dbConfig.MaxConnIdleTime

	dbPool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, err
	}

	return dbPool, nil
}

func setupRedis(cfg *config.Config) (*redis.Client, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		MaxRetries:   cfg.Redis.MaxRetries,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return redisClient, nil
}

func setupRouter(
	shortenerHandler *handler.ShortenerHandler,
	analyticsHandler *handler.AnalyticsHandler,
	healthHandler *handler.HealthHandler,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(middleware.Logger())

	// health check
	router.GET("/healthz", healthHandler.Healthz)
	router.GET("/readyz", healthHandler.Readyz)

	api := router.Group("/api")
	{
		api.POST("/shorten", shortenerHandler.ShortenURL)

		api.GET("/analytics/:shortCode", analyticsHandler.GetAnalytics)
		api.GET("/analytics/:shortCode/clicks", analyticsHandler.GetClickHistory)
	}

	router.GET("/:shortCode", shortenerHandler.Redirect)

	return router
}

func gracefulShutdown(srv *http.Server, timeout time.Duration, dbPool *pgxpool.Pool, redisClient *redis.Client, log *slog.Logger) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Info("Shutdown signal received", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Forced shutdown", "error", err)
	}

	dbPool.Close()
	log.Info("Database connection closed")

	if err := redisClient.Close(); err != nil {
		log.Error("Error closing Redis", "error", err)
	}

	log.Info("Graceful shutdown completed")
}
