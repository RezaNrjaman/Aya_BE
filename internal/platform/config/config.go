package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	AppEnv             string
	DatabaseURL        string
	JWTSecret          string
	JWTExpirationHours int
	ServiceKey         string
	AllowedOrigins     []string
}

// Load reads configuration from environment variables, optionally reading from .env file first.
func Load() (*Config, error) {
	// Try loading .env file if it exists, ignore error if file not present
	_ = godotenv.Load()

	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		AppEnv:             getEnv("APP_ENV", "development"),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		JWTSecret:          getEnv("JWT_SECRET", "dev_secret_key_minimum_32_characters_long_for_security"),
		JWTExpirationHours: getEnvInt("JWT_EXPIRATION_HOURS", 8),
		ServiceKey:         getEnv("SERVICE_KEY", "dev_inter_service_key"),
		AllowedOrigins:     splitComma(getEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173,*")),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks mandatory configuration according to environment.
func (c *Config) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("PORT is required")
	}

	if c.IsProduction() {
		if c.DatabaseURL == "" {
			return fmt.Errorf("DATABASE_URL is mandatory in production environment")
		}
		if len(c.JWTSecret) < 32 {
			return fmt.Errorf("JWT_SECRET must be at least 32 characters long in production")
		}
		if c.ServiceKey == "" || c.ServiceKey == "dev_inter_service_key" {
			return fmt.Errorf("SERVICE_KEY must be securely defined in production")
		}
	}

	return nil
}

func (c *Config) IsProduction() bool {
	return strings.ToLower(c.AppEnv) == "production"
}

func (c *Config) IsDevelopment() bool {
	return strings.ToLower(c.AppEnv) == "development"
}

func getEnv(key, defaultVal string) string {
	if val, exists := os.LookupEnv(key); exists && val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if valStr, exists := os.LookupEnv(key); exists && valStr != "" {
		if val, err := strconv.Atoi(valStr); err == nil {
			return val
		}
	}
	return defaultVal
}

func splitComma(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			res = append(res, trimmed)
		}
	}
	return res
}
