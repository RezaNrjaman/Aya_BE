package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	defaultJWTSecret           = "dev_secret_key_minimum_32_characters_long_for_security"
	defaultServiceKey          = "dev_inter_service_key"
	defaultAllowedOrigins      = "http://localhost:3000,http://localhost:5173"
	minimumSecretLength        = 32
	requiredJWTExpirationHours = 8
)

// Config contains validated runtime settings for the API server.
type Config struct {
	Port               string
	AppEnv             string
	DatabaseURL        string
	JWTSecret          string
	JWTExpirationHours int
	ServiceKey         string
	AllowedOrigins     []string
}

// Load reads environment configuration and validates it before startup.
func Load() (*Config, error) {
	if err := loadDotEnv(); err != nil {
		return nil, err
	}
	cfg, err := readEnvironment()
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadDotEnv() error {
	err := godotenv.Load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("load .env: %w", err)
	}
	return nil
}

func readEnvironment() (*Config, error) {
	expiration, err := getEnvInt("JWT_EXPIRATION_HOURS", 8)
	if err != nil {
		return nil, err
	}
	return &Config{
		Port: getEnv("PORT", "8080"), AppEnv: getEnv("APP_ENV", "development"),
		DatabaseURL: getEnv("DATABASE_URL", ""), JWTSecret: getEnv("JWT_SECRET", defaultJWTSecret),
		JWTExpirationHours: expiration, ServiceKey: getEnv("SERVICE_KEY", defaultServiceKey),
		AllowedOrigins: splitComma(getEnv("ALLOWED_ORIGINS", defaultAllowedOrigins)),
	}, nil
}

// Validate rejects incomplete or unsafe runtime configuration.
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("configuration is required")
	}
	if err := validateEnvironment(c.AppEnv); err != nil {
		return err
	}
	if err := validateRuntime(c); err != nil {
		return err
	}
	return validateDeployment(c)
}

func validateRuntime(c *Config) error {
	if err := validatePort(c.Port); err != nil {
		return err
	}
	if c.JWTExpirationHours != requiredJWTExpirationHours {
		return errors.New("JWT_EXPIRATION_HOURS must be 8 according to the platform contract")
	}
	if strings.TrimSpace(c.JWTSecret) == "" || strings.TrimSpace(c.ServiceKey) == "" {
		return errors.New("JWT_SECRET and SERVICE_KEY are required")
	}
	return nil
}

func validateDeployment(c *Config) error {
	secure := c.IsProductionLike()
	if err := validateDatabaseURL(c.DatabaseURL, secure); err != nil {
		return err
	}
	if err := validateSecrets(c, secure); err != nil {
		return err
	}
	return validateOrigins(c.AllowedOrigins, secure)
}

func validateEnvironment(environment string) error {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "development", "staging", "production", "test":
		return nil
	default:
		return errors.New("APP_ENV must be development, staging, production, or test")
	}
}

func validatePort(raw string) error {
	port, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || port < 1 || port > 65535 {
		return errors.New("PORT must be an integer between 1 and 65535")
	}
	return nil
}

func validateDatabaseURL(raw string, required bool) error {
	if strings.TrimSpace(raw) == "" {
		if required {
			return errors.New("DATABASE_URL is required in staging and production")
		}
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || !isPostgresURL(parsed) {
		return errors.New("DATABASE_URL must be a valid PostgreSQL connection URL")
	}
	if required && !hasSecureDatabaseCredentials(parsed) {
		return errors.New("DATABASE_URL must contain non-placeholder credentials")
	}
	return nil
}

func isPostgresURL(parsed *url.URL) bool {
	if parsed == nil || parsed.Hostname() == "" || parsed.User == nil {
		return false
	}
	if parsed.User.Username() == "" {
		return false
	}
	return strings.EqualFold(parsed.Scheme, "postgres") ||
		strings.EqualFold(parsed.Scheme, "postgresql")
}

func hasSecureDatabaseCredentials(parsed *url.URL) bool {
	password, exists := parsed.User.Password()
	if !exists || strings.TrimSpace(password) == "" {
		return false
	}
	return !containsPlaceholder(password) &&
		!containsPlaceholder(parsed.User.Username())
}

func validateSecrets(c *Config, required bool) error {
	if !required {
		return nil
	}
	if !isStrongSecret(c.JWTSecret) || c.JWTSecret == defaultJWTSecret {
		return errors.New("JWT_SECRET must be a unique secret of at least 32 characters")
	}
	if !isStrongSecret(c.ServiceKey) || c.ServiceKey == defaultServiceKey {
		return errors.New("SERVICE_KEY must be a unique secret of at least 32 characters")
	}
	return nil
}

func isStrongSecret(secret string) bool {
	value := strings.TrimSpace(secret)
	if value != secret || len(value) < minimumSecretLength ||
		strings.ContainsAny(value, " \t\r\n") {
		return false
	}
	if !hasMinimumCharacterVariety(value) {
		return false
	}
	return !containsPlaceholder(value)
}

func hasMinimumCharacterVariety(value string) bool {
	seen := make(map[rune]struct{})
	for _, character := range value {
		seen[character] = struct{}{}
		if len(seen) >= 8 {
			return true
		}
	}
	return false
}

func containsPlaceholder(secret string) bool {
	value := strings.ToLower(secret)
	markers := []string{"change-me", "change-before-production", "dev-only", "local-only",
		"replace-me", "your-secret", "placeholder", "example", "your_local_password"}
	for _, marker := range markers {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func validateOrigins(origins []string, requireHTTPS bool) error {
	if len(origins) == 0 {
		return errors.New("ALLOWED_ORIGINS must contain at least one origin")
	}
	for _, origin := range origins {
		if err := validateOrigin(origin, requireHTTPS); err != nil {
			return err
		}
	}
	return nil
}

func validateOrigin(origin string, requireHTTPS bool) error {
	value := strings.TrimSpace(origin)
	parsed, err := url.Parse(value)
	if err != nil || !isValidOrigin(parsed, value) {
		return errors.New("ALLOWED_ORIGINS contains an invalid origin")
	}
	if requireHTTPS && !strings.EqualFold(parsed.Scheme, "https") {
		return errors.New("staging and production origins must use HTTPS")
	}
	return nil
}

func isValidOrigin(parsed *url.URL, raw string) bool {
	if raw == "" || raw == "*" || parsed == nil || parsed.Hostname() == "" {
		return false
	}
	if !strings.EqualFold(parsed.Scheme, "http") &&
		!strings.EqualFold(parsed.Scheme, "https") {
		return false
	}
	return parsed.User == nil && parsed.Path == "" && parsed.RawQuery == "" &&
		!parsed.ForceQuery && parsed.Fragment == ""
}

// IsProduction reports whether the application runs in production.
func (c *Config) IsProduction() bool {
	return strings.EqualFold(strings.TrimSpace(c.AppEnv), "production")
}

// IsProductionLike reports whether the environment requires production safeguards.
func (c *Config) IsProductionLike() bool {
	return c.IsProduction() || strings.EqualFold(strings.TrimSpace(c.AppEnv), "staging")
}

// IsDevelopment reports whether the application runs in development.
func (c *Config) IsDevelopment() bool {
	return strings.EqualFold(strings.TrimSpace(c.AppEnv), "development")
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	return parsed, nil
}

func splitComma(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			values = append(values, value)
		}
	}
	return values
}
