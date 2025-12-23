package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gamassss/url-shortener/internal/config"
	"github.com/gamassss/url-shortener/internal/handler"
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

	dbPool, err := setupDatabase(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer dbPool.Close()

	redisClient, err := setupRedis(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer redisClient.Close()

	urlRepo := postgres.NewURLRepository(dbPool)
	urlCache := redisRepo.NewURLCache(redisClient)

	shortenerService := service.NewShortenerService(urlRepo, urlCache)

	shortenerHandler := handler.NewShortenerHandler(shortenerService)
	healthHandler := handler.NewHealthHandler(dbPool, redisClient)

	router := setupRouter(shortenerHandler, healthHandler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Printf("Server starting on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	gracefulShutdown(srv, cfg.Server.ShutdownTimeout, dbPool, redisClient)
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

func setupRouter(shortenerHandler *handler.ShortenerHandler, healthHandler *handler.HealthHandler) *gin.Engine {
	router := gin.Default()

	// health check
	router.GET("/healthz", healthHandler.Healthz)
	router.GET("/readyz", healthHandler.Readyz)

	api := router.Group("/api")
	{
		api.POST("/shorten", shortenerHandler.ShortenURL)
	}

	router.GET("/:shortCode", shortenerHandler.Redirect)

	return router
}

func gracefulShutdown(srv *http.Server, timeout time.Duration, dbPool *pgxpool.Pool, redisClient *redis.Client) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// block until received
	sig := <-quit
	log.Printf("Received: %v. Starting graceful shutdown...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	dbPool.Close()

	if err := redisClient.Close(); err != nil {
		log.Printf("Error closing Redis: %v", err)
	}

	log.Println("Graceful shutdown completed")
}
