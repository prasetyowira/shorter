package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port        int
	DatabaseURL string
	AuthUser    string
	AuthPass    string
	BaseURL     string
	CacheSize   int
	LogLevel    string
}

func LoadConfig() Config {
	port, _ := strconv.Atoi(getEnv("PORT", "8080"))
	cacheSize, _ := strconv.Atoi(getEnv("CACHE_SIZE", "1000"))

	return Config{
		Port:        port,
		DatabaseURL: getEnv("DATABASE_URL", "shorter.db"),
		AuthUser:    getEnv("AUTH_USER", "admin"),
		AuthPass:    getEnv("AUTH_PASS", "password"),
		BaseURL:     getEnv("BASE_URL", "http://localhost:8080"),
		CacheSize:   cacheSize,
		LogLevel:    getEnv("LOG_LEVEL", "INFO"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
} 