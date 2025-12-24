package config

import (
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Redis    RedisConfig
	Server   ServerConfig
	Database DatabaseConfig
	Log      LogConfig
}

type RedisConfig struct {
	Host         string
	Port         string
	Password     string
	DB           int
	Addr         string
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
}

type ServerConfig struct {
	Port            string
	ShutdownTimeout time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	URL             string
	MaxConns        int
	MinConns        int
	ConnMaxLifetime time.Duration
	MaxConnIdleTime time.Duration
}

type LogConfig struct {
	Level      string
	Format     string
	OutputPath string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("SERVER_SHUTDOWN_TIMEOUT", 30)
	viper.SetDefault("SERVER_READ_TIMEOUT", 10)
	viper.SetDefault("SERVER_WRITE_TIMEOUT", 10)

	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", 0)
	viper.SetDefault("REDIS_POOL_SIZE", 100)
	viper.SetDefault("REDIS_MIN_IDLE_CONNS", 20)
	viper.SetDefault("REDIS_MAX_RETRIES", 3)

	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "root")
	viper.SetDefault("DB_PASSWORD", "root")
	viper.SetDefault("DB_NAME", "urlshortener")
	viper.SetDefault("DB_MAX_CONNS", 40)
	viper.SetDefault("DB_MIN_CONNS", 10)
	viper.SetDefault("DB_CONN_MAX_LIFETIME", 5)
	viper.SetDefault("DB_CONN_MAX_IDLE_TIME", 30) // in seconds

	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_FORMAT", "json")
	viper.SetDefault("LOG_OUTPUT_PATH", "")
	viper.SetDefault("LOG_MAX_SIZE", 100)
	viper.SetDefault("LOG_MAX_BACKUPS", 3)
	viper.SetDefault("LOG_MAX_AGE", 7)
	viper.SetDefault("LOG_COMPRESS", true)

	if err := viper.ReadInConfig(); err != nil {
		log.Println("Warning: .env file not found, using default values")
	}

	redisConfig := RedisConfig{
		Host:         viper.GetString("REDIS_HOST"),
		Port:         viper.GetString("REDIS_PORT"),
		Password:     viper.GetString("REDIS_PASSWORD"),
		DB:           viper.GetInt("REDIS_DB"),
		PoolSize:     viper.GetInt("REDIS_POOL_SIZE"),
		MinIdleConns: viper.GetInt("REDIS_MIN_IDLE_CONNS"),
		MaxRetries:   viper.GetInt("REDIS_MAX_RETRIES"),
	}

	redisConfig.Addr = fmt.Sprintf("%s:%s", redisConfig.Host, redisConfig.Port)

	dbConfig := DatabaseConfig{
		Host:            viper.GetString("DB_HOST"),
		Port:            viper.GetString("DB_PORT"),
		User:            viper.GetString("DB_USER"),
		Password:        viper.GetString("DB_PASSWORD"),
		Name:            viper.GetString("DB_NAME"),
		MaxConns:        viper.GetInt("DB_MAX_CONNS"),
		MinConns:        viper.GetInt("DB_MIN_CONNS"),
		ConnMaxLifetime: time.Duration(viper.GetInt("DB_CONN_MAX_LIFETIME")) * time.Minute,
		MaxConnIdleTime: time.Duration(viper.GetInt("DB_CONN_MAX_IDLE_TIME")) * time.Second,
	}

	dbConfig.URL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbConfig.User,
		dbConfig.Password,
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.Name,
	)

	logConfig := LogConfig{
		Level:      viper.GetString("LOG_LEVEL"),
		Format:     viper.GetString("LOG_FORMAT"),
		OutputPath: viper.GetString("LOG_OUTPUT_PATH"),
		MaxSize:    viper.GetInt("LOG_MAX_SIZE"),
		MaxBackups: viper.GetInt("LOG_MAX_BACKUPS"),
		MaxAge:     viper.GetInt("LOG_MAX_AGE"),
		Compress:   viper.GetBool("LOG_COMPRESS"),
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:            viper.GetString("SERVER_PORT"),
			ShutdownTimeout: time.Duration(viper.GetInt("SERVER_SHUTDOWN_TIMEOUT")) * time.Second,
			ReadTimeout:     time.Duration(viper.GetInt("SERVER_READ_TIMEOUT")) * time.Second,
			WriteTimeout:    time.Duration(viper.GetInt("SERVER_WRITE_TIMEOUT")) * time.Second,
		},
		Redis:    redisConfig,
		Database: dbConfig,
		Log:      logConfig,
	}

	return cfg, nil
}
