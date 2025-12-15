package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Redis    RedisConfig
	Server   ServerConfig
	Database DatabaseConfig
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	Addr     string
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	URL      string
	MaxConns int
	MinConns int
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	viper.SetDefault("SERVER_PORT", "8080")

	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", 0)

	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "root")
	viper.SetDefault("DB_PASSWORD", "root")
	viper.SetDefault("DB_NAME", "urlshortener")
	viper.SetDefault("DB_MAX_CONNS", 100)
	viper.SetDefault("DB_MIN_CONNS", 10)

	if err := viper.ReadInConfig(); err != nil {
		log.Println("Warning: .env file not found, using default values")
	}

	redisConfig := RedisConfig{
		Host:     viper.GetString("REDIS_HOST"),
		Port:     viper.GetString("REDIS_PORT"),
		Password: viper.GetString("REDIS_PASSWORD"),
		DB:       viper.GetInt("REDIS_DB"),
	}

	redisConfig.Addr = fmt.Sprintf("%s:%s", redisConfig.Host, redisConfig.Port)

	dbConfig := DatabaseConfig{
		Host:     viper.GetString("DB_HOST"),
		Port:     viper.GetString("DB_PORT"),
		User:     viper.GetString("DB_USER"),
		Password: viper.GetString("DB_PASSWORD"),
		Name:     viper.GetString("DB_NAME"),
		MaxConns: viper.GetInt("DB_MAX_CONNS"),
		MinConns: viper.GetInt("DB_MIN_CONNS"),
	}

	dbConfig.URL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&pool_max_conns=%d&pool_min_conns=%d",
		dbConfig.User,
		dbConfig.Password,
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.Name,
		dbConfig.MaxConns,
		dbConfig.MinConns,
	)

	cfg := &Config{
		Server: ServerConfig{
			Port: viper.GetString("SERVER_PORT"),
		},
		Redis:    redisConfig,
		Database: dbConfig,
	}

	return cfg, nil
}
