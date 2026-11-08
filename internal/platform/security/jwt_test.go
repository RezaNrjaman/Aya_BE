package security_test

import (
	"testing"
	"time"

	"araya-business-hub/be/internal/platform/security"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "my_super_secret_test_key_minimum_32_bytes_long"

func TestGenerateAndValidateToken(t *testing.T) {
	unitID := 3
	claims := security.PlatformClaims{
		UserID:      42,
		Name:        "Rani Kusumawati",
		Role:        "kasir",
		Division:    "penjualan",
		UnitUsahaID: &unitID,
	}

	tokenStr, err := security.GenerateToken(claims, testSecret, 8)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	parsed, err := security.ValidateToken(tokenStr, testSecret)
	if err != nil {
		t.Fatalf("failed to validate valid token: %v", err)
	}

	if parsed.UserID != 42 || parsed.Name != "Rani Kusumawati" || parsed.Role != "kasir" {
		t.Fatalf("claims mismatch: %+v", parsed)
	}
	if parsed.UnitUsahaID == nil || *parsed.UnitUsahaID != 3 {
		t.Fatalf("expected unit_usaha_id = 3, got %v", parsed.UnitUsahaID)
	}
}

func TestValidateToken_Expired(t *testing.T) {
	now := time.Now().Add(-2 * time.Hour)
	claims := security.PlatformClaims{
		UserID:   1,
		Name:     "Admin",
		Role:     "admin",
		Division: "it",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(testSecret))

	_, err := security.ValidateToken(tokenStr, testSecret)
	if err != security.ErrTokenExpired {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	claims := security.PlatformClaims{
		UserID:   1,
		Name:     "Admin",
		Role:     "admin",
		Division: "it",
	}

	tokenStr, err := security.GenerateToken(claims, testSecret, 8)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Validate with a different secret
	_, err = security.ValidateToken(tokenStr, "another_secret_key_which_does_not_match_32_bytes")
	if err != security.ErrInvalidSignature && err != security.ErrTokenInvalid {
		t.Fatalf("expected signature error, got %v", err)
	}
}

func TestValidateToken_MissingClaims(t *testing.T) {
	// Missing UserID and Role
	claims := security.PlatformClaims{
		Name:     "NoID",
		Division: "it",
	}

	tokenStr, err := security.GenerateToken(claims, testSecret, 8)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = security.ValidateToken(tokenStr, testSecret)
	if err != security.ErrMissingClaims {
		t.Fatalf("expected ErrMissingClaims, got %v", err)
	}
}
