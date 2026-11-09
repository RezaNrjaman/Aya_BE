package config

import (
	"strings"
	"testing"
)

func validProductionConfig() *Config {
	return &Config{
		Port: "8080", AppEnv: "production",
		DatabaseURL:        "postgres://user:S3cure-DB-Pa55w0rd!@localhost:5432/araya",
		JWTSecret:          "unique-production-jwt-secret-with-more-than-32-characters",
		JWTExpirationHours: 8, ServiceKey: "unique-production-service-key-with-more-than-32-characters",
		AllowedOrigins: []string{"https://app.example.com"},
	}
}

func TestValidateAcceptsProductionConfig(t *testing.T) {
	if err := validProductionConfig().Validate(); err != nil {
		t.Fatalf("expected valid production config, got %v", err)
	}
}

type configTestCase struct {
	name   string
	mutate func(*Config)
	want   string
}

func invalidProductionCases() []configTestCase {
	return []configTestCase{
		{"environment", func(c *Config) { c.AppEnv = "prod" }, "APP_ENV"},
		{"port", func(c *Config) { c.Port = "70000" }, "PORT"},
		{"expiration low", func(c *Config) { c.JWTExpirationHours = 0 }, "JWT_EXPIRATION_HOURS"},
		{"expiration high", func(c *Config) { c.JWTExpirationHours = 12 }, "JWT_EXPIRATION_HOURS"},
		{"database missing", func(c *Config) { c.DatabaseURL = "" }, "DATABASE_URL"},
		{"database scheme", func(c *Config) { c.DatabaseURL = "mysql://user:pass@host/db" }, "PostgreSQL"},
		{"database placeholder", func(c *Config) { c.DatabaseURL = "postgres://user:YOUR_LOCAL_PASSWORD@host/db" }, "credentials"},
		{"weak secret", func(c *Config) { c.JWTSecret = strings.Repeat("a", 40) }, "JWT_SECRET"},
		{"jwt placeholder", func(c *Config) { c.JWTSecret = "dev-only-change-before-production-JWT-0123456789abcdef" }, "JWT_SECRET"},
		{"service key", func(c *Config) { c.ServiceKey = defaultServiceKey }, "SERVICE_KEY"},
		{"service placeholder", func(c *Config) { c.ServiceKey = "dev-only-change-before-production-SVC-0123456789abcdef" }, "SERVICE_KEY"},
		{"origin missing", func(c *Config) { c.AllowedOrigins = nil }, "ALLOWED_ORIGINS"},
		{"origin wildcard", func(c *Config) { c.AllowedOrigins = []string{"*"} }, "invalid origin"},
		{"origin http", func(c *Config) { c.AllowedOrigins = []string{"http://app.example.com"} }, "HTTPS"},
		{"staging origin", func(c *Config) { c.AppEnv = "staging"; c.AllowedOrigins = []string{"http://app.example.com"} }, "HTTPS"},
		{"origin path", func(c *Config) { c.AllowedOrigins = []string{"https://app.example.com/path"} }, "invalid origin"},
	}
}

func TestValidateRejectsInvalidConfiguration(t *testing.T) {
	for _, test := range invalidProductionCases() {
		t.Run(test.name, func(t *testing.T) {
			cfg := validProductionConfig()
			test.mutate(cfg)
			assertConfigError(t, cfg, test.want)
		})
	}
}

func TestValidateAllowsDevelopmentDefaults(t *testing.T) {
	cfg := &Config{
		Port: "8080", AppEnv: "development", JWTSecret: defaultJWTSecret,
		JWTExpirationHours: 8, ServiceKey: defaultServiceKey,
		AllowedOrigins: []string{"http://localhost:3000", "http://localhost:5173"},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid development config, got %v", err)
	}
}

func TestValidateAllowsDevelopmentWithoutDatabase(t *testing.T) {
	cfg := &Config{Port: "8080", AppEnv: "development", JWTSecret: defaultJWTSecret,
		JWTExpirationHours: 8, ServiceKey: defaultServiceKey,
		AllowedOrigins: []string{"http://localhost:5173"}}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected database to be optional in development, got %v", err)
	}
}

func TestGetEnvIntRejectsInvalidValue(t *testing.T) {
	t.Setenv("ARAYA_TEST_INT", "not-a-number")
	if _, err := getEnvInt("ARAYA_TEST_INT", 8); err == nil {
		t.Fatal("expected invalid integer error")
	}
}

func TestSplitCommaTrimsAndDropsEmptyValues(t *testing.T) {
	got := splitComma(" https://a.example, ,https://b.example ")
	if len(got) != 2 || got[0] != "https://a.example" || got[1] != "https://b.example" {
		t.Fatalf("unexpected split result: %#v", got)
	}
}

func assertConfigError(t *testing.T, cfg *Config, expected string) {
	t.Helper()
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q validation error, got %v", expected, err)
	}
}
