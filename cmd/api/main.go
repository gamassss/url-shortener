package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gamassss/url-shortener/internal/config"
	"github.com/gamassss/url-shortener/internal/handler"
	"github.com/gamassss/url-shortener/internal/repository/postgres"
	"github.com/gamassss/url-shortener/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
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

	urlRepo := postgres.NewURLRepository(dbPool)
	shortenerService := service.NewShortenerService(urlRepo)
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
