package main

import (
	"context"
	"fmt"
	"log"

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

	dbPool, err := pgxpool.New(context.Background(), cfg.Database.URL)
	if err != nil {
		log.Fatal(err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if err = redisClient.Ping(context.Background()).Err(); err != nil {
		fmt.Errorf("failed to connect to redis: %w", err)
	}

	defer redisClient.Close()

	urlRepo := postgres.NewURLRepository(dbPool)
	urlCache := redisRepo.NewURLCache(redisClient)
	shortenerService := service.NewShortenerService(urlRepo, urlCache)
	shortenerHandler := handler.NewShortenerHandler(shortenerService)

	router := gin.Default()

	api := router.Group("/api")
	{
		api.POST("/shorten", shortenerHandler.ShortenURL)
	}

	router.GET("/:shortCode", shortenerHandler.Redirect)

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err = router.Run(addr); err != nil {
		log.Fatal(err)
	}
}
