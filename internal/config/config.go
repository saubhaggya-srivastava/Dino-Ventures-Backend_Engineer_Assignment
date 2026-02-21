package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the wallet service
type Config struct {
	// Database configuration
	DatabaseURL     string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration

	// Server configuration
	ServerPort int

	// Business logic configuration
	IdempotencyTTL time.Duration
}

// LoadConfig loads configuration from environment variables with sensible defaults
func LoadConfig() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables or defaults")
	}

	return &Config{
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://localhost/wallet_service?sslmode=disable"),
		MaxOpenConns:    getEnvInt("MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvInt("MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: getEnvDuration("CONN_MAX_LIFETIME", 5*time.Minute),
		ConnMaxIdleTime: getEnvDuration("CONN_MAX_IDLE_TIME", 5*time.Minute),
		ServerPort:      getEnvInt("SERVER_PORT", 8080),
		IdempotencyTTL:  getEnvDuration("IDEMPOTENCY_TTL", 24*time.Hour),
	}
}

// Helper functions for environment variable parsing
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
